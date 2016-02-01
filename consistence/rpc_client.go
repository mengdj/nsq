package consistence

import (
	"github.com/absolute8511/nsq/nsqd"
	"net"
	"net/rpc"
	"time"
)

const (
	RPC_TIMEOUT       = time.Duration(time.Second * 10)
	RPC_TIMEOUT_SHORT = time.Duration(time.Second)
)

type NsqdRpcClient struct {
	remote     string
	timeout    time.Duration
	connection *rpc.Client
}

type RpcCallError struct {
	CallErr  error
	ReplyErr *CoordErr
}

func (self *RpcCallError) Error() string {
	errStr := ""
	if self.CallErr != nil {
		errStr += self.CallErr.Error()
	}
	if self.ReplyErr != nil {
		errStr += self.ReplyErr.Error()
	}
	return errStr
}

func NewNsqdRpcClient(addr string, timeout time.Duration) (*NsqdRpcClient, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}

	return &NsqdRpcClient{
		remote:     addr,
		timeout:    timeout,
		connection: rpc.NewClient(conn),
	}, nil
}

func (self *NsqdRpcClient) Reconnect() error {
	conn, err := net.DialTimeout("tcp", self.remote, self.timeout)
	if err != nil {
		return err
	}
	self.connection.Close()
	self.connection = rpc.NewClient(conn)
	return nil
}

func (self *NsqdRpcClient) CallWithRetry(method string, arg interface{}, reply interface{}) error {
	for {
		err := self.connection.Call(method, arg, reply)
		if err == rpc.ErrShutdown {
			err = self.Reconnect()
			if err != nil {
				return err
			}
		} else {
			coordLog.Infof("rpc call %v error: %v", method, err)
			return err
		}
	}
}

func (self *NsqdRpcClient) NotifyTopicLeaderSession(epoch int, topicInfo *TopicPartionMetaInfo, leaderSession *TopicLeaderSession) *RpcCallError {
	var rpcInfo RpcTopicLeaderSession
	rpcInfo.LookupdEpoch = epoch
	rpcInfo.TopicLeaderSession = *leaderSession
	rpcInfo.TopicName = topicInfo.Name
	rpcInfo.TopicPartition = topicInfo.Partition
	rpcInfo.TopicSession = leaderSession.Session
	rpcInfo.TopicLeaderEpoch = leaderSession.LeaderEpoch
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordinator.NotifyTopicLeaderSession", rpcInfo, &retErr)
	return &RpcCallError{err, &retErr}
}

func (self *NsqdRpcClient) UpdateTopicInfo(epoch int, topicInfo *TopicPartionMetaInfo) *RpcCallError {
	var rpcInfo RpcAdminTopicInfo
	rpcInfo.LookupdEpoch = epoch
	rpcInfo.TopicPartionMetaInfo = *topicInfo
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordinator.UpdateTopicInfo", rpcInfo, &retErr)
	return &RpcCallError{err, &retErr}
}

func (self *NsqdRpcClient) EnableTopicWrite(epoch int, topicInfo *TopicPartionMetaInfo) *RpcCallError {
	var rpcInfo RpcAdminTopicInfo
	rpcInfo.LookupdEpoch = epoch
	rpcInfo.TopicPartionMetaInfo = *topicInfo
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordinator.EnableTopicWrite", rpcInfo, &retErr)
	return &RpcCallError{err, &retErr}
}

func (self *NsqdRpcClient) DisableTopicWrite(epoch int, topicInfo *TopicPartionMetaInfo) *RpcCallError {
	var rpcInfo RpcAdminTopicInfo
	rpcInfo.LookupdEpoch = epoch
	rpcInfo.TopicPartionMetaInfo = *topicInfo
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordinator.DisableTopicWrite", rpcInfo, &retErr)
	return &RpcCallError{err, &retErr}
}

func (self *NsqdRpcClient) GetTopicStats(topic string) (*NodeTopicStats, error) {
	var stat NodeTopicStats
	err := self.CallWithRetry("NsqdCoordinator.GetTopicStats", topic, &stat)
	return &stat, err
}

func (self *NsqdRpcClient) UpdateCatchupForTopic(epoch int, info *TopicPartionMetaInfo) *RpcCallError {
	var rpcReq RpcAdminTopicInfo
	rpcReq.TopicPartionMetaInfo = *info
	rpcReq.LookupdEpoch = epoch
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordinator.UpdateCatchupForTopic", rpcReq, &retErr)
	return &RpcCallError{err, &retErr}
}

func (self *NsqdRpcClient) UpdateChannelsForTopic(epoch int, info *TopicPartionMetaInfo) *RpcCallError {
	var rpcReq RpcAdminTopicInfo
	rpcReq.TopicPartionMetaInfo = *info
	rpcReq.LookupdEpoch = epoch
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordinator.UpdateChannelsForTopic", rpcReq, &retErr)
	return &RpcCallError{err, &retErr}
}

func (self *NsqdRpcClient) UpdateChannelOffset(leaderEpoch int32, info *TopicPartionMetaInfo, channel string, offset ChannelConsumerOffset) *RpcCallError {
	var updateInfo RpcChannelOffsetArg
	updateInfo.TopicName = info.Name
	updateInfo.TopicPartition = info.Partition
	updateInfo.TopicEpoch = info.Epoch
	updateInfo.TopicLeaderEpoch = leaderEpoch
	updateInfo.Channel = channel
	updateInfo.ChannelOffset = offset
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordinator.UpdateChannelOffset", updateInfo, &retErr)
	return &RpcCallError{err, &retErr}
}

func (self *NsqdRpcClient) PutMessage(leaderEpoch int32, info *TopicPartionMetaInfo, log CommitLogData, message *nsqd.Message) *RpcCallError {
	var putData RpcPutMessage
	putData.LogData = log
	putData.TopicName = info.Name
	putData.TopicPartition = info.Partition
	putData.TopicMessage = message
	putData.TopicEpoch = info.Epoch
	putData.TopicLeaderEpoch = leaderEpoch
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordinator.PutMessage", putData, &retErr)
	return &RpcCallError{err, &retErr}
}

func (self *NsqdRpcClient) PutMessages(leaderEpoch int32, info *TopicPartionMetaInfo, loglist []CommitLogData, messages []*nsqd.Message) *RpcCallError {
	var putData RpcPutMessages
	putData.LogList = loglist
	putData.TopicName = info.Name
	putData.TopicPartition = info.Partition
	putData.TopicMessages = messages
	putData.TopicEpoch = info.Epoch
	putData.TopicLeaderEpoch = leaderEpoch
	var retErr CoordErr
	err := self.CallWithRetry("NsqdCoordinator.PutMessages", putData, &retErr)
	return &RpcCallError{err, &retErr}
}

func (self *NsqdRpcClient) GetLastCommmitLogID(topicInfo *TopicPartionMetaInfo) (int64, error) {
	var req RpcCommitLogReq
	req.TopicName = topicInfo.Name
	req.TopicPartition = topicInfo.Partition
	var ret int64
	err := self.CallWithRetry("NsqdCoordinator.GetLastCommitLogID", req, &ret)
	return ret, err
}

func (self *NsqdRpcClient) GetCommmitLogFromOffset(topicInfo *TopicPartionMetaInfo, offset int64) (int64, CommitLogData, error) {
	var req RpcCommitLogReq
	req.LogOffset = offset
	req.TopicName = topicInfo.Name
	req.TopicPartition = topicInfo.Partition
	var rsp RpcCommitLogRsp
	err := self.CallWithRetry("NsqdCoordinator.GetCommmitLogFromOffset", req, &rsp)
	return rsp.LogOffset, rsp.LogData, err
}

func (self *NsqdRpcClient) PullCommitLogsAndData(topic string, partition int,
	startOffset int64, num int) ([]CommitLogData, [][]byte, error) {
	var r RpcPullCommitLogsReq
	r.TopicName = topic
	r.TopicPartition = partition
	r.StartLogOffset = startOffset
	r.LogMaxNum = num
	var ret RpcPullCommitLogsRsp
	err := self.CallWithRetry("NsqdCoordinator.PullCommitLogs", r, &ret)
	return ret.Logs, ret.DataList, err
}