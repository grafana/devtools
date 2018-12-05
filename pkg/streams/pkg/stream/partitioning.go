package stream

import "fmt"

type PartitionKey interface {
	Get(key string) (interface{}, bool)
	GetKeys() []string
	GetValues() []interface{}
	GetCompositeKey() string
}

type partitionKey struct {
	sortedKeys []string
	values     map[string]interface{}
}

func newPartitionKey() *partitionKey {
	return &partitionKey{
		sortedKeys: []string{},
		values:     map[string]interface{}{},
	}
}

func (pk *partitionKey) Add(key string, value interface{}) {
	pk.sortedKeys = append(pk.sortedKeys, key)
	pk.values[key] = value
}

func (pk *partitionKey) Update(key string, value interface{}) {
	pk.values[key] = value
}

func (pk *partitionKey) Get(key string) (interface{}, bool) {
	if v, exists := pk.values[key]; exists {
		return v, true
	}

	return nil, false
}

func (pk *partitionKey) GetKeys() []string {
	keys := []string{}
	for _, key := range pk.sortedKeys {
		keys = append(keys, key)
	}
	return keys
}

func (pk *partitionKey) GetValues() []interface{} {
	values := []interface{}{}
	for _, key := range pk.sortedKeys {
		values = append(values, pk.values[key])
	}
	return values
}

func (pk *partitionKey) GetCompositeKey() string {
	str := ""
	for _, key := range pk.sortedKeys {
		if len(str) > 0 {
			str = str + "|"
		}
		str = str + fmt.Sprintf("%s", pk.values[key])
	}
	return str
}

type MessagePartition struct {
	Key      PartitionKey
	Messages []Msg
}

func NewMessagePartition(key PartitionKey) *MessagePartition {
	return &MessagePartition{
		Key:      key,
		Messages: []Msg{},
	}
}

func (mp *MessagePartition) Add(msg Msg) {
	mp.Messages = append(mp.Messages, msg)
}
