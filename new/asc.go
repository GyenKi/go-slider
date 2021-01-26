package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
)

// 私钥生成
//openssl genrsa -out rsa_private_key.pem 1024
var privateKey = []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDcGsUIIAINHfRTdMmgGwLrjzfMNSrtgIf4EGsNaYwmC1GjF/bM
h0Mcm10oLhNrKNYCTTQVGGIxuc5heKd1gOzb7bdTnCDPPZ7oV7p1B9Pud+6zPaco
qDz2M24vHFWYY2FbIIJh8fHhKcfXNXOLovdVBE7Zy682X1+R1lRK8D+vmQIDAQAB
AoGAeWAZvz1HZExca5k/hpbeqV+0+VtobMgwMs96+U53BpO/VRzl8Cu3CpNyb7HY
64L9YQ+J5QgpPhqkgIO0dMu/0RIXsmhvr2gcxmKObcqT3JQ6S4rjHTln49I2sYTz
7JEH4TcplKjSjHyq5MhHfA+CV2/AB2BO6G8limu7SheXuvECQQDwOpZrZDeTOOBk
z1vercawd+J9ll/FZYttnrWYTI1sSF1sNfZ7dUXPyYPQFZ0LQ1bhZGmWBZ6a6wd9
R+PKlmJvAkEA6o32c/WEXxW2zeh18sOO4wqUiBYq3L3hFObhcsUAY8jfykQefW8q
yPuuL02jLIajFWd0itjvIrzWnVmoUuXydwJAXGLrvllIVkIlah+lATprkypH3Gyc
YFnxCTNkOzIVoXMjGp6WMFylgIfLPZdSUiaPnxby1FNM7987fh7Lp/m12QJAK9iL
2JNtwkSR3p305oOuAz0oFORn8MnB+KFMRaMT9pNHWk0vke0lB1sc7ZTKyvkEJW0o
eQgic9DvIYzwDUcU8wJAIkKROzuzLi9AvLnLUrSdI6998lmeYO9x7pwZPukz3era
zncjRK3pbVkv0KrKfczuJiRlZ7dUzVO0b6QJr8TRAA==
-----END RSA PRIVATE KEY-----
`)

// 公钥: 根据私钥生成
//openssl rsa -in rsa_private_key.pem -pubout -out rsa_public_key.pem
var publicKey = []byte(`
-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDcGsUIIAINHfRTdMmgGwLrjzfM
NSrtgIf4EGsNaYwmC1GjF/bMh0Mcm10oLhNrKNYCTTQVGGIxuc5heKd1gOzb7bdT
nCDPPZ7oV7p1B9Pud+6zPacoqDz2M24vHFWYY2FbIIJh8fHhKcfXNXOLovdVBE7Z
y682X1+R1lRK8D+vmQIDAQAB
-----END PUBLIC KEY-----
`)

// 图片信息
// type SliderInfo struct {
// 	BacW    int    `json:"BacW"`
// 	BacH    int    `json:"BacH"`
// 	SliderW int    `json:"SliderW"`
// 	SliderH int    `json:"SliderH"`
// 	Dx      int    `json:"Dx"`
// 	Dy      int    `json:"Dy"`
// 	Src     string `json:"Src"`
// }
// func main() {
// 	slider := SliderInfo{
// 		BacW:    200,        // 背景图宽度
// 		BacH:    100,        // 背景图高度
// 		SliderW: 40,         // 滑块宽度
// 		SliderH: 40,         // 滑块高度
// 		Dx:      10,         // 滑块位置x坐标
// 		Dy:      10,         // 滑块位置y坐标
// 		Src:     "pic3.png", //图片地址
// 	}
// 	source, err := json.Marshal(slider)
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	fmt.Println("原字符：", string(source))
// 	key := "ABCDEFGHIJKLMNO1" //16位

// 	// 电码本模式（Electronic Codebook Book (ECB)）、
// 	encryptCodeECB := AESEncryptECB([]byte(source), []byte(key))
// 	fmt.Println("ECB密文：", string(encryptCodeECB))
// 	decryptCodeECB := AESDecryptECB(encryptCodeECB, []byte(key))
// 	fmt.Println("ECB解密：", string(decryptCodeECB))

// 	// 密码分组链接模式（Cipher Block Chaining (CBC)）
// 	encryptCodeCBC := AesEncryptCBC(string(source), key)
// 	fmt.Println("CBC密文：", encryptCodeCBC)
// 	decryptCodeCBC := AesDecryptCBC(encryptCodeCBC, key)
// 	fmt.Println("CBC解密结果：", decryptCodeCBC)

// 	decrs := SliderInfo{}
// 	err = json.Unmarshal([]byte(decryptCodeCBC), &decrs)
// 	if err != nil {
// 		log.Fatalln(err)
// 	}
// 	fmt.Printf("%+v\n", decrs)

// 	// 计算器模式（Counter (CTR)）
// 	encryptCodeCRT, _ := aesCtrCryptCRT([]byte(source), []byte(key))
// 	fmt.Println("CRT密文：", string(encryptCodeCRT))
// 	decryptCodeCRT, _ := aesCtrCryptCRT(encryptCodeCRT, []byte(key))
// 	fmt.Println("CRT解密：", string(decryptCodeCRT))

// 	// 密码反馈模式（Cipher FeedBack (CFB)）
// 	encryptCodeCFB := AesEncryptCFB([]byte(source), []byte(key))
// 	fmt.Println("CFB密文：", hex.EncodeToString(encryptCodeCFB))
// 	decryptCodeCFB := AesDecryptCFB(encryptCodeCFB, []byte(key))
// 	fmt.Println("CFB解密：", string(decryptCodeCFB))

// 	// 输出反馈模式（Output FeedBack (OFB)）
// 	encryptCodeOFB, _ := aesEncryptOFB([]byte(source), []byte(key))
// 	fmt.Println("OFB密文：", hex.EncodeToString(encryptCodeOFB))
// 	decryptCodeOFB, _ := aesDecryptOFB(encryptCodeOFB, []byte(key))
// 	fmt.Println("OFB解密：", string(decryptCodeOFB))

// 	// RSA加密
// 	encryptCodeRSA, _ := RsaEncrypt([]byte(string(source)))
// 	fmt.Println("RSA密文：", base64.StdEncoding.EncodeToString(encryptCodeRSA))
// 	decryptCodeRSA, _ := RsaDecrypt(encryptCodeRSA)
// 	fmt.Println("RSA解密：", string(decryptCodeRSA))
// }

func AESEncryptECB(src []byte, key []byte) (encrypted []byte) {
	cipher, _ := aes.NewCipher(generateKey(key))
	length := (len(src) + aes.BlockSize) / aes.BlockSize
	plain := make([]byte, length*aes.BlockSize)
	copy(plain, src)
	pad := byte(len(plain) - len(src))
	for i := len(src); i < len(plain); i++ {
		plain[i] = pad
	}
	encrypted = make([]byte, len(plain))
	// 分组分块加密
	for bs, be := 0, cipher.BlockSize(); bs <= len(src); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Encrypt(encrypted[bs:be], plain[bs:be])
	}

	return encrypted
}

func AESDecryptECB(encrypted []byte, key []byte) (decrypted []byte) {
	cipher, _ := aes.NewCipher(generateKey(key))
	decrypted = make([]byte, len(encrypted))
	//
	for bs, be := 0, cipher.BlockSize(); bs < len(encrypted); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Decrypt(decrypted[bs:be], encrypted[bs:be])
	}

	trim := 0
	if len(decrypted) > 0 {
		trim = len(decrypted) - int(decrypted[len(decrypted)-1])
	}

	return decrypted[:trim]
}

func generateKey(key []byte) (genKey []byte) {
	genKey = make([]byte, 16)
	copy(genKey, key)
	for i := 16; i < len(key); {
		for j := 0; j < 16 && i < len(key); j, i = j+1, i+1 {
			genKey[j] ^= key[i]
		}
	}
	return genKey
}

func AesEncryptCFB(origData []byte, key []byte) (encrypted []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		//panic(err)
	}
	encrypted = make([]byte, aes.BlockSize+len(origData))
	iv := encrypted[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		//panic(err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(encrypted[aes.BlockSize:], origData)
	return encrypted
}
func AesDecryptCFB(encrypted []byte, key []byte) (decrypted []byte) {
	block, _ := aes.NewCipher(key)
	if len(encrypted) < aes.BlockSize {
		panic("ciphertext too short")
	}
	iv := encrypted[:aes.BlockSize]
	encrypted = encrypted[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(encrypted, encrypted)
	return encrypted
}

// 加密
func RsaEncrypt(origData []byte) ([]byte, error) {
	//解密pem格式的公钥
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, errors.New("public key error")
	}
	// 解析公钥
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	// 类型断言
	pub := pubInterface.(*rsa.PublicKey)
	//加密
	return rsa.EncryptPKCS1v15(rand.Reader, pub, origData)
}

// 解密
func RsaDecrypt(ciphertext []byte) ([]byte, error) {
	//解密
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, errors.New("private key error!")
	}
	//解析PKCS1格式的私钥
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	// 解密
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
}

func aesEncryptOFB(data []byte, key []byte) ([]byte, error) {
	data = PKCS7Padding(data, aes.BlockSize)
	block, _ := aes.NewCipher([]byte(key))
	out := make([]byte, aes.BlockSize+len(data))
	iv := out[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewOFB(block, iv)
	stream.XORKeyStream(out[aes.BlockSize:], data)
	return out, nil
}

func aesDecryptOFB(data []byte, key []byte) ([]byte, error) {
	block, _ := aes.NewCipher([]byte(key))
	iv := data[:aes.BlockSize]
	data = data[aes.BlockSize:]
	if len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("data is not a multiple of the block size")
	}

	out := make([]byte, len(data))
	mode := cipher.NewOFB(block, iv)
	mode.XORKeyStream(out, data)

	out = PKCS7UnPadding(out)
	return out, nil
}

//加密
func aesCtrCryptCRT(plainText []byte, key []byte) ([]byte, error) {

	//1. 创建cipher.Block接口
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//2. 创建分组模式，在crypto/cipher包中
	iv := bytes.Repeat([]byte("1"), block.BlockSize())
	stream := cipher.NewCTR(block, iv)
	//3. 加密
	dst := make([]byte, len(plainText))
	stream.XORKeyStream(dst, plainText)

	return dst, nil
}

//key随意设置 data 要加密数据
func Hmac(key, data string) string {
	hash := hmac.New(md5.New, []byte(key)) // 创建对应的md5哈希加密算法
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum([]byte("")))
}
func HmacSha256(key, data string) string {
	hash := hmac.New(sha256.New, []byte(key)) //创建对应的sha256哈希加密算法
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum([]byte("")))
}

func AesEncryptCBC(orig string, key string) string {
	// 转成字节数组
	origData := []byte(orig)
	k := []byte(key)
	// 分组秘钥
	// NewCipher该函数限制了输入k的长度必须为16, 24或者32
	block, _ := aes.NewCipher(k)
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 补全码
	origData = PKCS7Padding(origData, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])
	// 创建数组
	cryted := make([]byte, len(origData))
	// 加密
	blockMode.CryptBlocks(cryted, origData)
	return base64.StdEncoding.EncodeToString(cryted)
}
func AesDecryptCBC(cryted string, key string) string {
	// 转成字节数组
	crytedByte, _ := base64.StdEncoding.DecodeString(cryted)
	k := []byte(key)
	// 分组秘钥
	block, _ := aes.NewCipher(k)
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])
	// 创建数组
	orig := make([]byte, len(crytedByte))
	// 解密
	blockMode.CryptBlocks(orig, crytedByte)
	// 去补全码
	orig = PKCS7UnPadding(orig)
	return string(orig)
}

//补码
//AES加密数据块分组长度必须为128bit(byte[16])，密钥长度可以是128bit(byte[16])、192bit(byte[24])、256bit(byte[32])中的任意一个。
func PKCS7Padding(ciphertext []byte, blocksize int) []byte {
	padding := blocksize - len(ciphertext)%blocksize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

//去码
func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
