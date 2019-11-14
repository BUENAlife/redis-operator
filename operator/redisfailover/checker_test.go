package redisfailover_test

import (
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spotahome/redis-operator/log"
	"github.com/spotahome/redis-operator/metrics"
	mRFService "github.com/spotahome/redis-operator/mocks/operator/redisfailover/service"
	mK8SService "github.com/spotahome/redis-operator/mocks/service/k8s"
	rfOperator "github.com/spotahome/redis-operator/operator/redisfailover"
)

func TestCheckAndHeal(t *testing.T) {
	tests := []struct {
		name                           string
		nMasters                       int
		nRedis                         int
		forceNewMaster                 bool
		slavesOK                       bool
		sentinelMonitorOK              bool
		sentinelNumberInMemoryOK       bool
		sentinelSlavesNumberInMemoryOK bool
	}{
		{
			name:                           "Everything ok, no need to heal",
			nMasters:                       1,
			nRedis:                         3,
			forceNewMaster:                 false,
			slavesOK:                       true,
			sentinelMonitorOK:              true,
			sentinelNumberInMemoryOK:       true,
			sentinelSlavesNumberInMemoryOK: true,
		},
		{
			name:                           "Multiple masters",
			nMasters:                       2,
			nRedis:                         3,
			forceNewMaster:                 false,
			slavesOK:                       true,
			sentinelMonitorOK:              true,
			sentinelNumberInMemoryOK:       true,
			sentinelSlavesNumberInMemoryOK: true,
		},
		{
			name:                           "No masters but wait",
			nMasters:                       0,
			nRedis:                         3,
			forceNewMaster:                 false,
			slavesOK:                       true,
			sentinelMonitorOK:              true,
			sentinelNumberInMemoryOK:       true,
			sentinelSlavesNumberInMemoryOK: true,
		},
		{
			name:                           "No masters, only one redis available, make master",
			nMasters:                       0,
			nRedis:                         1,
			forceNewMaster:                 false,
			slavesOK:                       true,
			sentinelMonitorOK:              true,
			sentinelNumberInMemoryOK:       true,
			sentinelSlavesNumberInMemoryOK: true,
		},
		{
			name:                           "No masters, set random",
			nMasters:                       0,
			nRedis:                         3,
			forceNewMaster:                 true,
			slavesOK:                       true,
			sentinelMonitorOK:              true,
			sentinelNumberInMemoryOK:       true,
			sentinelSlavesNumberInMemoryOK: true,
		},
		{
			name:                           "Slaves from master wrong",
			nMasters:                       1,
			nRedis:                         3,
			forceNewMaster:                 false,
			slavesOK:                       false,
			sentinelMonitorOK:              true,
			sentinelNumberInMemoryOK:       true,
			sentinelSlavesNumberInMemoryOK: true,
		},
		{
			name:                           "Sentinels not pointing correct monitor",
			nMasters:                       1,
			nRedis:                         3,
			forceNewMaster:                 false,
			slavesOK:                       true,
			sentinelMonitorOK:              false,
			sentinelNumberInMemoryOK:       true,
			sentinelSlavesNumberInMemoryOK: true,
		},
		{
			name:                           "Sentinels with wrong number of sentinels",
			nMasters:                       1,
			nRedis:                         3,
			forceNewMaster:                 false,
			slavesOK:                       true,
			sentinelMonitorOK:              true,
			sentinelNumberInMemoryOK:       false,
			sentinelSlavesNumberInMemoryOK: true,
		},
		{
			name:                           "Sentinels with wrong number of slaves",
			nMasters:                       1,
			nRedis:                         3,
			forceNewMaster:                 false,
			slavesOK:                       true,
			sentinelMonitorOK:              true,
			sentinelNumberInMemoryOK:       true,
			sentinelSlavesNumberInMemoryOK: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			rf := generateRF(false)

			expErr := false
			continueTests := true

			master := "0.0.0.0"
			sentinel := "1.1.1.1"

			config := generateConfig()
			mk := &mK8SService.Services{}
			mrfs := &mRFService.RedisFailoverClient{}
			mrfc := &mRFService.RedisFailoverCheck{}
			mrfh := &mRFService.RedisFailoverHeal{}
			mrfc.On("CheckRedisNumber", rf).Once().Return(nil)
			mrfc.On("CheckSentinelNumber", rf).Once().Return(nil)
			mrfc.On("GetNumberMasters", rf).Once().Return(test.nMasters, nil)
			switch test.nMasters {
			case 0:
				mrfc.On("GetRedisesIPs", rf).Once().Return(make([]string, test.nRedis), nil)
				if test.nRedis == 1 {
					mrfh.On("MakeMaster", mock.Anything, rf).Once().Return(nil)
					break
				}
				if test.forceNewMaster {
					mrfc.On("GetMinimumRedisPodTime", rf).Once().Return(1*time.Hour, nil)
					mrfh.On("SetOldestAsMaster", rf).Once().Return(nil)
				} else {
					mrfc.On("GetMinimumRedisPodTime", rf).Once().Return(1*time.Second, nil)
					continueTests = false
				}
			case 1:
				break
			default:
				expErr = true
			}
			if !expErr && continueTests {
				mrfc.On("GetMasterIP", rf).Once().Return(master, nil)
				if test.slavesOK {
					mrfc.On("CheckAllSlavesFromMaster", master, rf).Once().Return(nil)
				} else {
					mrfc.On("CheckAllSlavesFromMaster", master, rf).Once().Return(errors.New(""))
					mrfh.On("SetMasterOnAll", master, rf).Once().Return(nil)
				}
				mrfc.On("GetRedisesIPs", rf).Twice().Return([]string{master}, nil)
				mrfc.On("CheckRedisSyncing", master, rf).Once().Return(false, nil)
				mrfc.On("GetStatefulSetVersion", rf).Once().Return("1", nil)
				mrfc.On("GetRedisesSlavesPods", rf).Once().Return([]string{}, nil)
				mrfc.On("GetRedisesMasterPod", rf).Once().Return(master, nil)
				mrfc.On("GetRedisVersion", master, rf).Once().Return("1", nil)
				mrfh.On("SetRedisCustomConfig", master, rf).Once().Return(nil)
				mrfc.On("GetSentinelsIPs", rf).Once().Return([]string{sentinel}, nil)
				if test.sentinelMonitorOK {
					mrfc.On("CheckSentinelMonitor", sentinel, master).Once().Return(nil)
				} else {
					mrfc.On("CheckSentinelMonitor", sentinel, master).Once().Return(errors.New(""))
					mrfh.On("NewSentinelMonitor", sentinel, master, rf).Once().Return(nil)
				}
				if test.sentinelNumberInMemoryOK {
					mrfc.On("CheckSentinelNumberInMemory", sentinel, rf).Once().Return(nil)
				} else {
					mrfc.On("CheckSentinelNumberInMemory", sentinel, rf).Once().Return(errors.New(""))
					mrfh.On("RestoreSentinel", sentinel).Once().Return(nil)
				}
				if test.sentinelSlavesNumberInMemoryOK {
					mrfc.On("CheckSentinelSlavesNumberInMemory", sentinel, rf).Once().Return(nil)
				} else {
					mrfc.On("CheckSentinelSlavesNumberInMemory", sentinel, rf).Once().Return(errors.New(""))
					mrfh.On("RestoreSentinel", sentinel).Once().Return(nil)
				}
				mrfh.On("SetSentinelCustomConfig", sentinel, rf).Once().Return(nil)
			}

			handler := rfOperator.NewRedisFailoverHandler(config, mrfs, mrfc, mrfh, mk, metrics.Dummy, log.Dummy)
			err := handler.CheckAndHeal(rf)

			if expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
			mrfc.AssertExpectations(t)
			mrfh.AssertExpectations(t)
		})
	}
}

func TestUpdate(t *testing.T) {
	type podStatus struct {
		pod     corev1.Pod
		syncing bool
		master  bool
	}
	tests := []struct {
		name        string
		pods        []podStatus
		ssVersion   string
		errExpected error
		syncing     bool
	}{
		{
			name: "all ok, no change needed",
			pods: []podStatus{
				{
					pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "slave1",
							Labels: map[string]string{
								"deployment-version": "10",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "0.0.0.0",
						},
					},
					master:  false,
					syncing: false,
				},
				{
					pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "slave2",
							Labels: map[string]string{
								"deployment-version": "10",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "0.0.0.1",
						},
					},
					master:  false,
					syncing: false,
				},
				{
					pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "master",
							Labels: map[string]string{
								"deployment-version": "10",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "1.1.1.1",
						},
					},
					master:  true,
					syncing: false,
				},
			},
			ssVersion:   "10",
			errExpected: nil,
		},
		{
			name: "syncing",
			pods: []podStatus{
				{
					pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "slave1",
							Labels: map[string]string{
								"deployment-version": "10",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "0.0.0.0",
						},
					},
					master:  false,
					syncing: false,
				},
				{
					pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "slave2",
							Labels: map[string]string{
								"deployment-version": "10",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "0.0.0.1",
						},
					},
					master:  false,
					syncing: true,
				},
				{
					pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "master",
							Labels: map[string]string{
								"deployment-version": "10",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "1.1.1.1",
						},
					},
					master:  true,
					syncing: false,
				},
			},
			ssVersion:   "10",
			errExpected: nil,
		},
		{
			name: "pod version incorrect",
			pods: []podStatus{
				{
					pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "slave1",
							Labels: map[string]string{
								"deployment-version": "9",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "0.0.0.0",
						},
					},
					master:  false,
					syncing: false,
				},
				{
					pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "slave2",
							Labels: map[string]string{
								"deployment-version": "9",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "0.0.0.1",
						},
					},
					master:  false,
					syncing: true,
				},
				{
					pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "master",
							Labels: map[string]string{
								"deployment-version": "9",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "1.1.1.1",
						},
					},
					master:  true,
					syncing: false,
				},
			},
			ssVersion:   "10",
			errExpected: nil,
		},
		{
			name: "master version incorrect",
			pods: []podStatus{
				{
					pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "slave1",
							Labels: map[string]string{
								"deployment-version": "10",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "0.0.0.0",
						},
					},
					master:  false,
					syncing: false,
				},
				{
					pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "slave2",
							Labels: map[string]string{
								"deployment-version": "10",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "0.0.0.1",
						},
					},
					master:  false,
					syncing: true,
				},
				{
					pod: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "master",
							Labels: map[string]string{
								"deployment-version": "10",
							},
						},
						Status: corev1.PodStatus{
							PodIP: "1.1.1.1",
						},
					},
					master:  true,
					syncing: false,
				},
			},
			ssVersion:   "10",
			errExpected: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			rf := generateRF(false)

			config := generateConfig()
			mrfs := &mRFService.RedisFailoverClient{}

			mrfc := &mRFService.RedisFailoverCheck{}
			mrfc.On("GetRedisesIPs", rf).Once().Return([]string{"0.0.0.0", "0.0.0.1", "1.1.1.1"}, nil)

			next := true

			for _, pod := range test.pods {
				mrfc.On("CheckRedisSyncing", pod.pod.Status.PodIP, rf).Once().Return(pod.syncing, nil)
				if pod.syncing {
					next = false
					break
				}
			}
			mrfh := &mRFService.RedisFailoverHeal{}

			if next {
				mrfc.On("GetStatefulSetVersion", rf).Once().Return(test.ssVersion, nil)
				mrfc.On("GetRedisesSlavesPods", rf).Once().Return([]string{"slave1", "slave2"}, nil)

				for _, pod := range test.pods {
					mrfc.On("GetRedisVersion", pod.pod.ObjectMeta.Name, rf).Once().Return(pod.pod.ObjectMeta.Labels["deployment-version"], nil)
					if pod.pod.ObjectMeta.Labels["deployment-version"] != test.ssVersion {
						mrfh.On("DeletePod", mock.Anything, mock.Anything)
						if pod.master == false {
							next = false
							break
						}
					}
				}
				if next {
					mrfc.On("GetRedisesMasterPod", rf).Once().Return("master", nil)
				}
			}

			mk := &mK8SService.Services{}

			handler := rfOperator.NewRedisFailoverHandler(config, mrfs, mrfc, mrfh, mk, metrics.Dummy, log.Dummy)
			err := handler.UpdateRedisesPods(rf)

			if test.errExpected != nil {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}

			mrfc.AssertExpectations(t)
			mrfh.AssertExpectations(t)

		})
	}
}
