package messages

type (
	ExtendedMetadataRequestMsg struct {
		Piece     int                            `bencode:"piece"`
		TotalSize int                            `bencode:"total_size"`
		Type      ExtendedMetadataRequestMsgType `bencode:"msg_type"`
	}

	ExtendedMetadataRequestMsgType int
)

// Returns the expected piece size for this request message. This is needed to determine the offset
// into an extension message payload that the request metadata piece data starts.
func (me ExtendedMetadataRequestMsg) PieceSize() int {
	ret := me.TotalSize - me.Piece*(1<<14)
	if ret > 1<<14 {
		ret = 1 << 14
	}
	return ret
}
