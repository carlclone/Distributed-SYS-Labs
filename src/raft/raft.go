package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)
import "sync/atomic"
import "../labrpc"

// import "bytes"
// import "../labgob"

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

const (
	ROLE_FOLLOWER  = 1
	ROLE_CANDIDATE = 2
	ROLE_LEADER    = 3
)

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	//persistent
	currentTerm int
	votedFor    int
	//log         []*LogEntry

	//volatile
	commitIndex int
	lastApplied int

	//volatile / only leader

	//other
	lastTimeHeard   time.Time
	role            int
	electTimePeriod time.Duration

	//follower channels , 当做接收事件的通道
	electTimesUp        chan bool
	receivedHeartBeat   chan AppendEntriesArgs
	receivedRequestVote chan bool

	//candidate channels
	voteForSelf        chan bool
	becomeLeader       chan bool
	voteBeGranted      chan RequestVoteArgs
	rvReqsReceived     chan RequestVoteArgs
	rvReplyReceived    chan RequestVoteReply
	termLessThanOthers chan int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	term = rf.currentTerm
	if rf.role == ROLE_LEADER {
		isleader = true
	}
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term              int //当前 term
	LeaderId          int
	PrevLogIndex      int //
	PrevLogTerm       int
	Entries           []interface{}
	LeaderCommitIndex int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).

	rf.rvReqsReceived <- *args

	var tmpReply RequestVoteReply = <-rf.rvReplyReceived
	DPrintf("RequestVote")
	DPrintf(fmt.Sprint(tmpReply))
	DPrintf(strconv.Itoa(len(rf.peers)))
	reply.VoteGranted = tmpReply.VoteGranted
	//reply.Term = tmpReply.Term

	return
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {

	if args.Entries == nil {
		rf.receivedHeartBeat <- *args

		//if args.Term > rf.currentTerm {
		//	rf.termLessThanOthers <- args.Term
		//}

	}

	return
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) electTimeDetect() {
	//DPrintf("检测超时时间")
	//DPrintf(rf.electTimePeriod.String())
	//DPrintf(rf.lastTimeHeard.String())
	if rf.lastTimeHeard.Add(rf.electTimePeriod).Before(time.Now()) {

		rf.electTimesUp <- true
		rf.voteForSelf <- true
	}
}

//发送心跳包给所有人除了自己
func (rf *Raft) sendHeartBeats() {
	for i, _ := range rf.peers {
		if i == rf.me {
			continue
		}
		args := AppendEntriesArgs{}
		args.Term = rf.currentTerm
		args.LeaderId = rf.me

		reply := AppendEntriesReply{}
		rf.sendAppendEntries(i, &args, &reply)
	}
}

//发送RV给所有人除了自己
func (rf *Raft) askForVotes() {

	voteCount := 1
	DPrintf("给自己投票")
	for i, _ := range rf.peers {
		if i == rf.me {
			continue
		}
		args := RequestVoteArgs{}
		args.Term = rf.currentTerm
		args.CandidateId = rf.me

		reply := RequestVoteReply{}
		ok := rf.sendRequestVote(i, &args, &reply)
		var word string
		if ok {
			word = "成功"
		} else {
			word = "失败"
		}
		DPrintf("发送请求结果:" + strconv.Itoa(i) + word)
		if !ok {
			continue
		}
		if reply.VoteGranted == true {
			voteCount++
		}
	}
	DPrintf("投票结果" + strconv.Itoa(voteCount))
	if voteCount > len(rf.peers)/2 {
		rf.becomeLeader <- true
	}
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.electTimePeriod = time.Duration(rand.Int63()%400+250) * time.Millisecond
	rf.lastTimeHeard = time.Now()
	rf.role = ROLE_FOLLOWER
	rf.votedFor = -1

	rf.electTimesUp = make(chan bool, 50)

	rf.receivedHeartBeat = make(chan AppendEntriesArgs, 50)
	rf.receivedRequestVote = make(chan bool, 50)
	rf.voteForSelf = make(chan bool, 50)
	rf.becomeLeader = make(chan bool, 50)
	rf.voteBeGranted = make(chan RequestVoteArgs, 50)
	rf.rvReqsReceived = make(chan RequestVoteArgs, 50)
	rf.termLessThanOthers = make(chan int, 50)

	rf.rvReplyReceived = make(chan RequestVoteReply)

	DPrintf("创建peer")

	// Your initialization code here (2A, 2B, 2C).
	//定时事件触发线程    (在 rpc handler 里也会触发事件,小心并发race) ,
	// 如果我把所有 race 的逻辑通过推事件的形式执行,是不是就不用 lock 了
	go func() {
		for {

			switch rf.role {

			case ROLE_LEADER:
				rf.sendHeartBeats()
			}

			time.Sleep(1 * time.Millisecond)
		}
	}()

	go func() {
		for {

			switch rf.role {
			case ROLE_FOLLOWER:
				rf.electTimeDetect()
			}

			time.Sleep(1 * time.Millisecond)
		}
	}()

	//事件循环模型线程 , 每个角色的事件需要执行的逻辑
	go func() {
		// 是candidate , 需要发送RV , 并判断自己是否成为了leader ,
		for {

			//再检查一下 figure2 里的两个 RPC

			switch rf.role {
			//回应 c 的投票请求(要修改 voteFor) , l 的心跳请求
			//心跳和投票都会重置选举时间
			//选举超时投票
			case ROLE_FOLLOWER:
				select {

				case args := <-rf.receivedHeartBeat:
					//收到心跳包,重置选举超时时间
					handleFReceivedHeartBeat(rf, args)
				case args := <-rf.rvReqsReceived:
					handleFVoteReqs(rf, args)
				case <-rf.electTimesUp:
					handleFElectTimeUp(rf)
				}

			//currentTerm++ , vote++ , reset elect time , 汇集选票
			//获得大多数 , 成为 leader
			//收到心跳包 / AE RPC, 变成 follower
			//超时 , 开始新一轮
			case ROLE_CANDIDATE:
				select {
				case <-rf.voteForSelf:
					//给自己投票和汇集选票
					rf.currentTerm++
					rf.votedFor = me
					rf.askForVotes()
				case <-rf.electTimesUp:
					//超时,开始新一轮选举
					rf.voteForSelf <- true
				case <-rf.receivedHeartBeat:
					//变成 follower
					rf.role = ROLE_FOLLOWER
				case <-rf.becomeLeader:
					//变成leader
					DPrintf("变成leader")
					rf.role = ROLE_LEADER
					rf.sendHeartBeats()

				}

				//定期发心跳包
				//如果leader断开重连 , 收到了心跳包,则变成follower
			case ROLE_LEADER:

			}

			time.Sleep(1 * time.Millisecond)
		}

	}()

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	return rf
}

func handleFElectTimeUp(rf *Raft) {
	DPrintf("已超时" + strconv.Itoa(rf.me))
	//选举超时,变成 candidate
	//DPrintf(fmt.Sprint(rf.me))
	DPrintf("变成c")
	rf.role = ROLE_CANDIDATE
}

func handleFVoteReqs(rf *Raft, args RequestVoteArgs) {
	rf.lastTimeHeard = time.Now()
	DPrintf("重置超时")
	var reply = RequestVoteReply{}
	reply.Term = args.Term
	if args.Term < rf.currentTerm {
		reply.VoteGranted = false
	}
	DPrintf(fmt.Sprint("voteFor:" + strconv.Itoa(rf.votedFor)))
	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		//&& (args.LastLogIndex == rf.commitIndex && args.LastLogTerm == rf.currentTerm) {
		reply.VoteGranted = true

		rf.votedFor = args.CandidateId
		rf.currentTerm = args.Term
	}
	rf.rvReplyReceived <- reply
}

func handleFReceivedHeartBeat(rf *Raft, args AppendEntriesArgs) {
	//DPrintf("收到心跳包")
	rf.lastTimeHeard = time.Now()
	rf.currentTerm = args.Term
	rf.votedFor = -1
}
