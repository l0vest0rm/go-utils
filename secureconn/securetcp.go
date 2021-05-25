package secureconn

import (
	"io"
	"log"
	"net"
)

// 加密传输的 TCP Socket
type SecureTCPConn struct {
	io.ReadWriteCloser
	Cipher *Cipher
}

// 从输入流里读取加密过的数据，解密后把原数据放到bs里
func (secureSocket *SecureTCPConn) DecodeRead(bs []byte) (n int, err error) {
	n, err = secureSocket.Read(bs)
	if err != nil {
		return
	}
	secureSocket.Cipher.Decode(bs[:n])
	return
}

// 把放在bs里的数据加密后立即全部写入输出流
func (secureSocket *SecureTCPConn) EncodeWrite(bs []byte) (int, error) {
	secureSocket.Cipher.Encode(bs)
	return secureSocket.Write(bs)
}

// 从src中源源不断的读取原数据加密后写入到dst，直到src中没有数据可以再读取
func (secureSocket *SecureTCPConn) EncodeCopy(dst *SecureTCPConn) error {
	buf := BufferPoolGet()
	defer BufferPoolPut(buf)
	for {
		readCount, errRead := secureSocket.Read(buf)
		if errRead != nil {
			if errRead != io.EOF {
				return errRead
			} else {
				return nil
			}
		}
		if readCount > 0 {
			writeCount, errWrite := dst.EncodeWrite(buf[0:readCount])
			if errWrite != nil {
				return errWrite
			}
			if readCount != writeCount {
				return io.ErrShortWrite
			}
		}
	}
}

// 从src中源源不断的读取加密后的数据解密后写入到dst，直到src中没有数据可以再读取
func (secureSocket *SecureTCPConn) DecodeCopy(dst *SecureTCPConn) error {
	buf := BufferPoolGet()
	defer BufferPoolPut(buf)
	for {
		readCount, errRead := secureSocket.DecodeRead(buf)
		if errRead != nil {
			if errRead != io.EOF {
				return errRead
			} else {
				return nil
			}
		}
		if readCount > 0 {
			writeCount, errWrite := dst.Write(buf[0:readCount])
			if errWrite != nil {
				return errWrite
			}
			if readCount != writeCount {
				return io.ErrShortWrite
			}
		}
	}
}

// see net.DialTCP
func DialEncryptedTCP(raddr *net.TCPAddr) (*SecureTCPConn, error) {
	remoteConn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		return nil, err
	}
	// Conn被关闭时直接清除所有数据 不管没有发送的数据
	remoteConn.SetLinger(0)

	return &SecureTCPConn{
		ReadWriteCloser: remoteConn,
	}, nil
}

// see net.ListenTCP
func ListenEncryptedTCP(laddr *net.TCPAddr, handleConn func(localConn *SecureTCPConn), didListen func(listenAddr *net.TCPAddr)) error {
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	if didListen != nil {
		// didListen 可能有阻塞操作
		go didListen(listener.Addr().(*net.TCPAddr))
	}

	for {
		localConn, err := listener.AcceptTCP()
		if err != nil {
			log.Println(err)
			continue
		}
		// localConn被关闭时直接清除所有数据 不管没有发送的数据
		localConn.SetLinger(0)
		go handleConn(&SecureTCPConn{
			ReadWriteCloser: localConn,
		})
	}
}
