package rados

// #cgo LDFLAGS: -lrados
// #include <errno.h>
// #include <stdlib.h>
// #include <rados/librados.h>
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/ceph/go-ceph/errutil"
)

type RadosError int

func (e RadosError) Error() string {
	errno, s := errutil.FormatErrno(int(e))
	if s == "" {
		return fmt.Sprintf("rados: ret=%d", errno)
	}
	return fmt.Sprintf("rados: ret=%d, %s", errno, s)
}

var RadosAllNamespaces = C.LIBRADOS_ALL_NSPACES

var RadosErrorNotFound = RadosError(-C.ENOENT)
var RadosErrorPermissionDenied = RadosError(-C.EPERM)

func getRadosError(err int) error {
	if err == 0 {
		return nil
	}
	return RadosError(err)
}

// Version returns the major, minor, and patch components of the version of
// the RADOS library linked against.
func Version() (int, int, int) {
	var c_major, c_minor, c_patch C.int
	C.rados_version(&c_major, &c_minor, &c_patch)
	return int(c_major), int(c_minor), int(c_patch)
}

func makeConn() *Conn {
	return &Conn{connected: false}
}

func newConn(user *C.char) (*Conn, error) {
	conn := makeConn()
	ret := C.rados_create(&conn.cluster, user)

	if ret != 0 {
		return nil, RadosError(int(ret))
	}

	runtime.SetFinalizer(conn, freeConn)
	return conn, nil
}

// NewConn creates a new connection object. It returns the connection and an
// error, if any.
func NewConn() (*Conn, error) {
	return newConn(nil)
}

// NewConnWithUser creates a new connection object with a custom username.
// It returns the connection and an error, if any.
func NewConnWithUser(user string) (*Conn, error) {
	c_user := C.CString(user)
	defer C.free(unsafe.Pointer(c_user))
	return newConn(c_user)
}

// NewConnWithClusterAndUser creates a new connection object for a specific cluster and username.
// It returns the connection and an error, if any.
func NewConnWithClusterAndUser(clusterName string, userName string) (*Conn, error) {
	c_cluster_name := C.CString(clusterName)
	defer C.free(unsafe.Pointer(c_cluster_name))

	c_name := C.CString(userName)
	defer C.free(unsafe.Pointer(c_name))

	conn := makeConn()
	ret := C.rados_create2(&conn.cluster, c_cluster_name, c_name, 0)
	if ret != 0 {
		return nil, RadosError(int(ret))
	}

	runtime.SetFinalizer(conn, freeConn)
	return conn, nil
}

// freeConn releases resources that are allocated while configuring the
// connection to the cluster. rados_shutdown() should only be needed after a
// successful call to rados_connect(), however if the connection has been
// configured with non-default parameters, some of the parameters may be
// allocated before connecting. rados_shutdown() will free the allocated
// resources, even if there has not been a connection yet.
//
// This function is setup as a destructor/finalizer when rados_create() is
// called.
func freeConn(conn *Conn) {
	if conn.cluster != nil {
		C.rados_shutdown(conn.cluster)
		// prevent calling rados_shutdown() more than once
		conn.cluster = nil
	}
}
