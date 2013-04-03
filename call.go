package dbus

import (
	"encoding/binary"
	"errors"
	"strings"
)

// Call represents a pending or completed method call.
type Call struct {
	Destination string
	Path        ObjectPath
	Method      string
	Args        []interface{}

	// Strobes when the call is complete.
	Done chan *Call

	// After completion, the error status. If this is non-nil, it may be an
	// error message from the peer (with Error as its type) or some other error.
	Err error

	// Holds the response once the call is done. Structs are represented as
	// a slice of empty interfaces.
	Body []interface{}
}

var errSignature = errors.New("mismatched signature")

// Store stores the body of the reply into the provided pointers. It returns
// an error if the signatures of the body and retvalues don't match, or if
// the error status is not nil.
func (c *Call) Store(retvalues ...interface{}) error {
	if c.Err != nil {
		return c.Err
	}

	return Store(c.Body, retvalues...)
}

// Object represents a remote object on which methods can be invoked.
type Object struct {
	conn *Conn
	dest string
	path ObjectPath
}

// Call calls a method with (*Object).Go and waits for its reply.
func (o *Object) Call(method string, flags Flags, args ...interface{}) *Call {
	return <-o.Go(method, flags, make(chan *Call, 1), args...).Done
}

// Go calls a method with the given arguments asynchronously. It returns a
// Call structure representing this method call. The passed channel will
// return the same value once the call is done. If ch is nil, a new channel
// will be allocated. Otherwise, ch has to be buffered or Call will panic.
//
// If the flags include FlagNoReplyExpected, nil is returned and ch is ignored.
//
// If the method parameter contains a dot ('.'), the part before the last dot
// specifies the interface on which the method is called.
func (o *Object) Go(method string, flags Flags, ch chan *Call, args ...interface{}) *Call {
	iface := ""
	i := strings.LastIndex(method, ".")
	if i != -1 {
		iface = method[:i]
	}
	method = method[i+1:]
	msg := new(Message)
	msg.Order = binary.LittleEndian
	msg.Type = TypeMethodCall
	msg.serial = <-o.conn.serial
	msg.Flags = flags & (FlagNoAutoStart | FlagNoReplyExpected)
	msg.Headers = make(map[HeaderField]Variant)
	msg.Headers[FieldPath] = MakeVariant(o.path)
	msg.Headers[FieldDestination] = MakeVariant(o.dest)
	msg.Headers[FieldMember] = MakeVariant(method)
	if iface != "" {
		msg.Headers[FieldInterface] = MakeVariant(iface)
	}
	msg.Body = args
	if len(args) > 0 {
		msg.Headers[FieldSignature] = MakeVariant(GetSignature(args...))
	}
	if msg.Flags&FlagNoReplyExpected == 0 {
		if ch == nil {
			ch = make(chan *Call, 10)
		} else if cap(ch) == 0 {
			panic("(*dbus.Object).Go: unbuffered channel")
		}
		call := &Call{
			Destination: o.dest,
			Path:        o.path,
			Method:      method,
			Args:        args,
			Done:        ch,
		}
		o.conn.callsLck.Lock()
		o.conn.calls[msg.serial] = call
		o.conn.callsLck.Unlock()
		o.conn.out <- msg
		return call
	}
	o.conn.out <- msg
	return nil
}

// Destination returns the destination that calls on o are sent to.
func (o *Object) Destination() string {
	return o.dest
}

// Path returns the path that calls on o are sent to.
func (o *Object) Path() ObjectPath {
	return o.path
}
