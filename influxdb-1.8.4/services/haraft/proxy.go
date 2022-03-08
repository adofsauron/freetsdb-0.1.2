package haraft

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

)

func (s *Service) ProxyServiceOpen() error {
	s.Logger.Info("Open ProxyServiceOpen ready")

	bindAddress := g_localIp + ":" + g_proxyPort
	ln, err := net.Listen("tcp", bindAddress)
	if err != nil {
		s.Logger.Error(fmt.Sprintf("Open Server net.Listen fail, err = %v", err))
		return fmt.Errorf("ProxyServiceOpen listen fail, err = %v", err)
	}

	s.Listener = ln

	go s.ProxyServiceServe()

	s.Logger.Info(fmt.Sprintf("ProxyServiceOpen net.Listen ok, proxyAddr = %s", bindAddress))
	return nil
}

func (s *Service) ProxyServiceServe() {
	for {
		conn, err := s.Listener.Accept()
		if err, ok := err.(interface {
			Temporary() bool
		}); ok && err.Temporary() {
			continue
		}

		if nil != err {
			s.Logger.Error(fmt.Sprintf("ProxyServiceServe fail, Listener.Accept = %v", err))
			continue
		}

		go s.ProxyServiceHandle(conn)
	}
}

func (s *Service) ProxyServiceHandle(conn net.Conn) {
	defer conn.Close()

	for {
		b, err := io.ReadAll(conn)
		if nil != err {
			s.Logger.Error(fmt.Sprintf("ProxyServiceHandle fail, io.ReadFull err = %v", err))
			return
		}

		byteLen := len(b)
		if 0 == byteLen {
			s.Logger.Info(fmt.Sprintf("ProxyServiceHandle ignore, 0 == byteLen"))
			return
		}

		byte_apply := b[1:]
		s.Logger.Info(fmt.Sprintf("ProxyServiceHandle ready ApplyByte, byteLen = %d", byteLen))

		err = s.ProxyServiceProcessByte(byte_apply)
		if nil != err {
			s.Logger.Error(fmt.Sprintf("ProxyServiceHandle fail, ProxyServiceProcessByte err = %v", err))
			continue
		}
	}
}

func (s *Service) ProxyServiceProcessByte(b []byte) error {
	if len(b) < 8 {
		return fmt.Errorf("too short: len = %d", len(b))
	}

	var err error
	b_n := b[0:8]
	rf_type := int(binary.BigEndian.Uint64(b_n))

	s.Logger.Info(fmt.Sprintf("ProxyServiceProcessByte, rf_type: %d", rf_type))

	rf_type = (int)(rf_type)
	switch rf_type {
	case RFTYPE_WRITEPOINT:
		err = s.WritePointsPrivilegedToLeader(b)
	case RFTYPE_QUERY:
		err = s.ServeQueryToLeader(b)
	}

	return err
}
