/**
 * Copyright (c) 2022, Xerra Earth Observation Institute.
 * Copyright (c) 2025, Simeon Miteff.
 *
 * See LICENSE.TXT in the root directory of this source tree.
 */

package exporter

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"reflect"
)

// GetFdFromConnSafe extracts the file descriptor from a net.Conn,
// including support for *tls.Conn connections using simple reflection.
func GetFdFromConnSafe(conn net.Conn) (int, error) {
	if conn == nil {
		return -1, errors.New("connection is nil")
	}
	
	var fd int
	var err error
	
	defer func() {
		if r := recover(); r != nil {
			fd = -1
			err = fmt.Errorf("failed to get file descriptor: %v", r)
		}
	}()
	
	// For TLS connections, we need to first get the underlying connection
	if tlsConn, ok := conn.(*tls.Conn); ok {
		// Get the tls.Conn struct value
		v := reflect.Indirect(reflect.ValueOf(tlsConn))
		
		// Get the conn field (which contains the underlying net.Conn)
		connField := v.FieldByName("conn")
		if !connField.IsValid() {
			return -1, errors.New("failed to get conn field from tls.Conn")
		}
		
		// The conn field is an interface, get its element (the actual connection)
		// This will be the underlying *net.TCPConn in most cases
		underlyingConn := reflect.Indirect(connField.Elem())
		
		// Now follow the same path as regular TCP connections:
		// TCPConn -> conn (net.conn) -> fd (netFD) -> pfd (poll.FD) -> Sysfd
		netConn := underlyingConn.FieldByName("conn")
		netFD := reflect.Indirect(netConn).FieldByName("fd")
		pfd := reflect.Indirect(netFD).FieldByName("pfd")
		sysfd := pfd.FieldByName("Sysfd")
		
		fd = int(sysfd.Int())
	} else {
		// For regular connections, use the standard netfd approach
		v := reflect.Indirect(reflect.ValueOf(conn))
		connField := v.FieldByName("conn")
		netFD := reflect.Indirect(connField.FieldByName("fd"))
		pfd := netFD.FieldByName("pfd")
		fd = int(pfd.FieldByName("Sysfd").Int())
	}
	
	if err != nil {
		return -1, err
	}
	
	if fd < 0 {
		return -1, errors.New("invalid file descriptor")
	}
	
	return fd, nil
}