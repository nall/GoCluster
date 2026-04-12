package packets

import (
	"bytes"
	"testing"
)

func BenchmarkReadPacketPublish(b *testing.B) {
	packet := NewControlPacket(Publish).(*PublishPacket)
	packet.TopicName = "pskreporter/test"
	packet.Payload = []byte(`{"f":14074000,"md":"FT8","rp":5,"t":1712870400,"sc":"K1ABC","sl":"FN42","rc":"N0CALL","rl":"EM10"}`)

	var encoded bytes.Buffer
	if err := packet.Write(&encoded); err != nil {
		b.Fatalf("write publish packet: %v", err)
	}
	wireBytes := encoded.Bytes()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := ReadPacket(bytes.NewReader(wireBytes)); err != nil {
			b.Fatalf("read packet: %v", err)
		}
	}
}
