# schema.go分析

这里面的代码主要是给添加到数据库的key加上前缀后缀等。

```go
	// Data item prefixes (use single byte to avoid mixing data types, avoid `i`, used for indexes).
	// 数据项前缀（使用单字节以避免混合数据类型，避免使用“ i”作为索引）。
	headerPrefix       = []byte("h") // headerPrefix + num (uint64 big endian) + hash -> header
	headerTDSuffix     = []byte("t") // headerPrefix + num (uint64 big endian) + hash + headerTDSuffix -> td
	headerHashSuffix   = []byte("n") // headerPrefix + num (uint64 big endian) + headerHashSuffix -> hash
	headerNumberPrefix = []byte("H") // headerNumberPrefix + hash -> num (uint64 big endian)

	blockBodyPrefix     = []byte("b") // blockBodyPrefix + num (uint64 big endian) + hash -> block body
	blockReceiptsPrefix = []byte("r") // blockReceiptsPrefix + num (uint64 big endian) + hash -> block receipts

	txLookupPrefix  = []byte("l") // txLookupPrefix + hash -> transaction/receipt lookup metadata
	bloomBitsPrefix = []byte("B") // bloomBitsPrefix + bit (uint16 big endian) + section (uint64 big endian) + hash -> bloom bits

	preimagePrefix = []byte("secure-key-")      // preimagePrefix + hash -> preimage
	configPrefix   = []byte("ethereum-config-") // config prefix for the db

	// Chain index prefixes (use `i` + single byte to avoid mixing data types).
	// 链索引前缀（使用“ i” +单字节以避免混合数据类型）。
	BloomBitsIndexPrefix = []byte("iB") // BloomBitsIndexPrefix is the data table of a chain indexer to track its progress
```

添加前后缀的代码如下，

```go
// encodeBlockNumber encodes a block number as big endian uint64
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// headerKeyPrefix = headerPrefix + num (uint64 big endian)
func headerKeyPrefix(number uint64) []byte {
	return append(headerPrefix, encodeBlockNumber(number)...)
}

// headerKey = headerPrefix + num (uint64 big endian) + hash
func headerKey(number uint64, hash common.Hash) []byte {
	return append(append(headerPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// headerTDKey = headerPrefix + num (uint64 big endian) + hash + headerTDSuffix
func headerTDKey(number uint64, hash common.Hash) []byte {
	return append(headerKey(number, hash), headerTDSuffix...)
}

// headerHashKey = headerPrefix + num (uint64 big endian) + headerHashSuffix
func headerHashKey(number uint64) []byte {
	return append(append(headerPrefix, encodeBlockNumber(number)...), headerHashSuffix...)
}

// headerNumberKey = headerNumberPrefix + hash
func headerNumberKey(hash common.Hash) []byte {
	return append(headerNumberPrefix, hash.Bytes()...)
}

// blockBodyKey = blockBodyPrefix + num (uint64 big endian) + hash
func blockBodyKey(number uint64, hash common.Hash) []byte {
	return append(append(blockBodyPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// blockReceiptsKey = blockReceiptsPrefix + num (uint64 big endian) + hash
func blockReceiptsKey(number uint64, hash common.Hash) []byte {
	return append(append(blockReceiptsPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// txLookupKey = txLookupPrefix + hash
func txLookupKey(hash common.Hash) []byte {
	return append(txLookupPrefix, hash.Bytes()...)
}

// bloomBitsKey = bloomBitsPrefix + bit (uint16 big endian) + section (uint64 big endian) + hash
func bloomBitsKey(bit uint, section uint64, hash common.Hash) []byte {
	key := append(append(bloomBitsPrefix, make([]byte, 10)...), hash.Bytes()...)

	binary.BigEndian.PutUint16(key[1:], uint16(bit))
	binary.BigEndian.PutUint64(key[3:], section)

	return key
}

// preimageKey = preimagePrefix + hash
func preimageKey(hash common.Hash) []byte {
	return append(preimagePrefix, hash.Bytes()...)
}

// configKey = configPrefix + hash
func configKey(hash common.Hash) []byte {
	return append(configPrefix, hash.Bytes()...)
}
```

以下是底层键值数据库中，键值的对应情况，实际的value都是将其内容RLP编码后的结果

|                             key                              |                  value                  |
| :----------------------------------------------------------: | :-------------------------------------: |
| **headerPrefix**("h") + **num** (uint64 big endian) + **hash** |               **header**                |
| **headerPrefix**("h") + **num** (uint64 big endian) + **hash** + **headerTDSuffix**("t") |                 **td**                  |
| **headerPrefix**("h") + **num** (uint64 big endian) + **headerHashSuffix**("n") |                **hash**                 |
|            **headerNumberPrefix**("H") + **hash**            |       **num** (uint64 big endian)       |
| **blockBodyPrefix**("b") + **num** (uint64 big endian) + **hash** |             **block body**              |
| **blockReceiptsPrefix**("r") + **num** (uint64 big endian) + **hash** |           **block receipts**            |
|              **txLookupPrefix**("l") + **hash**              | **transaction/receipt lookup metadata** |
| **bloomBitsPrefix**("B") + **bit** (uint16 big endian) + **section** (uint64 big endian) + **hash** |             **bloom bits**              |
|         **preimagePrefix**("secure-key-") + **hash**         |              **preimage**               |
|       **configPrefix**("ethereum-config-") + **hash**        |               **config**                |

