package numericsapp

// NDataInt is a data structure that represents an integer for the Numerics iOS App
type NDataInt struct {
	Value int `json:"value"`
}

func NewNDataInt(v int) *NDataInt {
	return &NDataInt{
		Value: v,
	}
}
