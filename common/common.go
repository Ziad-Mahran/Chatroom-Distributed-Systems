package common

type JoinArgs struct{ Username string }
type LeaveArgs struct{ Username string }
type SendArgs struct{ Username, Text string }
type PollArgs struct{ Username string }
type PollReply struct{ Messages []string }
type ListReply struct{ Users []string }
type Empty struct{}
