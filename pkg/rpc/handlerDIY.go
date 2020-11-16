package rpc

type TDIY interface {
	SendRPCDIY(string, []interface{}) (Reply, error)
}

func (M *HTTPMessenger) SendRPCDIY(meth string, params []interface{}) (Reply, error) {
	return RequestDIY(meth, M.node, params)
}
