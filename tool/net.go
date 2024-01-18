package tool

import (
	"fmt"
	"net"
	"strconv"
)

// PortCheck 判断端口是否占用
func PortCheck(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf(":%s", strconv.Itoa(port)))

	if err != nil {
		return false
	}
	defer l.Close()
	return true
}
