package app

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"

	"go.uber.org/zap"
)

const (
	serialNumber = 1
	ip4GrayZone  = 127
	yearsGrant   = 1
	RSALen       = 4096
	CertsPerm    = 0600
)

func CreateCertificates(logger *zap.SugaredLogger) (privateKey *rsa.PrivateKey, certBytes []byte, err error) {
	// создаём шаблон сертификата
	cert := &x509.Certificate{
		// указываем уникальный номер сертификата
		SerialNumber: big.NewInt(serialNumber),
		// заполняем базовую информацию о владельце сертификата
		Subject: pkix.Name{
			Organization: []string{"GophKeeper"},
			Country:      []string{"RU"},
		},
		// разрешаем использование сертификата для 127.0.0.1 и ::1
		IPAddresses: []net.IP{net.IPv4(ip4GrayZone, 0, 0, 1), net.IPv6loopback},
		// сертификат верен, начиная со времени создания
		NotBefore: time.Now(),
		// время жизни сертификата — 10 лет
		NotAfter:     time.Now().AddDate(yearsGrant, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		// устанавливаем использование ключа для цифровой подписи,
		// а также серверной авторизации
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	// создаём новый приватный RSA-ключ длиной 4096 бит
	// обратите внимание, что для генерации ключа и сертификата
	// используется rand.Reader в качестве источника случайных данных
	privateKey, err = rsa.GenerateKey(rand.Reader, RSALen)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating RSA key: %w", err)
	}

	// создаём сертификат x.509
	certBytes, err = x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating certificat: %w", err)
	}

	return privateKey, certBytes, nil
}

func WriteCertificates(
	tlsCert []byte,
	tlsCertPath string,
	privateKey *rsa.PrivateKey,
	tlsKeyPath string,
	logger *zap.SugaredLogger,
) error {
	if err := os.Mkdir("certs", os.ModePerm); err != nil {
		return fmt.Errorf("unhandled mkdir to certs: %w", err)
	}

	// кодируем сертификат и ключ в формате PEM, который
	// используется для хранения и обмена криптографическими ключами
	certFile, err := os.OpenFile(tlsCertPath, os.O_WRONLY|os.O_CREATE, CertsPerm)
	if err != nil {
		return fmt.Errorf("error opening cert file: %w", err)
	}

	defer func() {
		if err := certFile.Close(); err != nil {
			logger.Error(err)
		}
	}()

	if err := pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: tlsCert,
	}); err != nil {
		return fmt.Errorf("error encoding cert file: %w", err)
	}

	rsaFile, err := os.OpenFile(tlsKeyPath, os.O_WRONLY|os.O_CREATE, CertsPerm)
	if err != nil {
		return fmt.Errorf("error opening key file: %w", err)
	}

	defer func() {
		if err := rsaFile.Close(); err != nil {
			logger.Error(err)
		}
	}()

	if err := pem.Encode(rsaFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}); err != nil {
		return fmt.Errorf("error encoding RSA private key: %w", err)
	}

	return nil
}
