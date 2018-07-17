package encoding

import (
	"bytes"
	"io"

	cbor "github.com/polydawn/refmt/cbor"
	"github.com/polydawn/refmt/obj/atlas"
)

type proxyWriter struct {
	w io.Writer
}

func (w *proxyWriter) Write(b []byte) (int, error) {
	return w.w.Write(b)
}

// Marshaller is a reusbale CBOR marshaller.
type Marshaller struct {
	marshal *cbor.Marshaller
	writer  proxyWriter
}

// NewMarshallerAtlased constructs a new cbor Marshaller using the given atlas.
func NewMarshallerAtlased(atl atlas.Atlas) *Marshaller {
	m := new(Marshaller)
	m.marshal = cbor.NewMarshallerAtlased(&m.writer, atl)
	return m
}

// Encode encodes the given object to the given writer.
func (m *Marshaller) Encode(obj interface{}, w io.Writer) error {
	m.writer.w = w
	err := m.marshal.Marshal(obj)
	m.writer.w = nil
	return err
}

// Marshal marshels the given object to a byte slice.
func (m *Marshaller) Marshal(obj interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := m.Encode(obj, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PooledMarshaller is a thread-safe pooled CBOR marshaller.
type PooledMarshaller struct {
	Count       int
	marshallers chan *Marshaller
}

// SetAtlas set sets the pool's atlas. It is *not* safe to call this
// concurrently.
func (p *PooledMarshaller) SetAtlas(atlas atlas.Atlas) {
	p.marshallers = make(chan *Marshaller, p.Count)
	for len(p.marshallers) < cap(p.marshallers) {
		p.marshallers <- NewMarshallerAtlased(atlas)
	}
}

// Marshal marshals the passed object using the pool of marshallers.
func (p *PooledMarshaller) Marshal(obj interface{}) ([]byte, error) {
	m := <-p.marshallers
	bts, err := m.Marshal(obj)
	p.marshallers <- m
	return bts, err
}

// Encode encodes the passed object to the given writer using the pool of
// marshallers.
func (p *PooledMarshaller) Encode(obj interface{}, w io.Writer) error {
	m := <-p.marshallers
	err := m.Encode(obj, w)
	p.marshallers <- m
	return err
}