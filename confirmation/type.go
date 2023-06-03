package confirmation

type Confirmation struct {
	ID      uint64
	Key     uint64
	OfferID uint64
}

type jsonObj = map[string]interface{}
