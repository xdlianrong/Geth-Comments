# Account
account包实现了以太坊客户端的钱包和账户管理。以太坊的钱包提供了KeyStore，usb两类钱包。同时以太坊合约的ABI的代码也在abi目录。

## Account.go
account.go定义了accounts模块对外导出的一些结构体和接口，包括Account结构体、Wallet接口和Backend接口。其中Account由一个以太坊地址和钱包路径组成；而各种类型的钱包需要实现Wallet和Backend接口来接入账入管理。

## hd.go
该代码定义了HD类型的钱包的路径解析等函数。这个文件中的注释还解析了HD路径一些知识。

## url.go
这个文件中的代码定义了代表以太坊钱包路径的URL结构体及相关函数。与hd.go不同的是URL结构体中保存了钱包的类型和钱包路径的字符串形式的表示，hd.go中的比较单一                                                            

## manager.go
这定义了Manager结构及其方法，这是account模块对外导出的主要的结构和方法之一。其它模块(比如cmd/geth中)通过这个结构体提供的方法对钱包进行管理

## errors.go
对一些错误的定义

# KeyStore
## account_cache.go
此文件中的代码实现了accountCache结构体及方法。accountCache的功能是在内存中缓存keystore钱包目录下所有账号信息。无论keystore目录中的文件如何变动（新建、删除、修改），accountCache都可以在扫描目录时将变动更新到内存中。

## file_cache.go
此文件中的代码实现了fileCache结构体及相关代码。与account_cache.go类似，file_cache.go中实现了对keystore目录下所有文件的信息的缓存。accountCache就是通过fileCache来获取文件变动的信息，进而得到账号变动信息的。

## key.go
key.go主要定义了Key结构体及其json格式的marshal/unmarshal方式。另外这个文件中还定义了通过keyStore接口将Key写入文件中的函数。keyStore接口中定义了Key被写入文件的具体细节，在passphrase.go和plain.go中都有实现。

## keystore.go
这个文件里的代码定义了KeyStore结构体及其方法。KeyStore结构体实现了Backend接口，是keystore类型的钱包的后端实现。同时它也实现了keystore类型钱包的大多数功能。

## passphrase.go
passphrase.go中定义了keyStorePassphrase结构体及其方法。keyStorePassphrase结构体是对keyStore接口（在key.go文件中）的一种实现方式，它会要求调用者提供一个密码，从而使用aes加密算法加密私钥后，将加密数据写入文件中。

## plain.go
这个文件中的代码定义了keyStorePlain结构体及其方法。keyStorePlain与keyStorePassphrase类似，也是对keyStore接口的实现。不同的是，keyStorePlain直接将密码明文存储在文件中。目前这种方式已被标记弃用且整个以太坊项目中都没有调用这个文件里的函数的地方，确实谁也不想将自己的私钥明文存在本地磁盘上。

## wallet.go
wallet.go中定义了keystoreWallet结构体及其方法。keystoreWallet是keystore类型的钱包的实现，但其功能基本都是调用KeyStore对象实现的。

## watch.go
watch.go中定义了watcher结构体及其方法。watcher用来监控keystore目录下的文件，如果文件发生变化，则立即调用account_cache.go中的代码重新扫描账户信息

# usbwallet
该文件为对硬件钱包的访问，我们可能不会用到，所以暂时可以不用关注

# scwallet
这个文件夹是关于不同account之间的互相安全通信（secure wallet），通过定义会话秘钥、二级秘钥来确保通话双方的信息真实、不被篡改、利用。 尤其是转账信息更不能被利用、被他人打开、和被篡改。

# abi
ABI是Application Binary Interface的缩写，字面意思 应用二进制接口，可以通俗的理解为合约的接口说明。当合约被编译后，那么它的abi也就确定了。abi主要是处理智能合约与账户的交互。

# HD：分层确定性钱包
是一种key的派生方式，它可以只使用一个公钥(称这个公钥为主公钥，其对应的私钥称为主私钥)的情况下，生成任意多个子公钥，而这些子公钥都是可以被主私钥控制的。每一个key都有自己的路径，即是是一个派生的key，这一点和keystore类型是一样的
(此处有个疑问，这是否与联盟链相关)
