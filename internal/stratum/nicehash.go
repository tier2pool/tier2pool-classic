package stratum

// https://github.com/nicehash/Specifications/blob/master/EthereumStratum_NiceHash_v1.0.0.txt

const (
	MethodNiceHashSubscribe = "mining.subscribe"
	MethodNiceHashNotify    = "mining.notify"
	MethodNiceHashSubmit    = "mining.submit"
	MethodNiceHashAuthorize = "mining.authorize"
)

type NiceHashAuthorizeParams []string

type NiceHashNotifyParams []any

type NiceHashSubmitParams []string
