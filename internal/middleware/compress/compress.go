package compress

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	contentEncoding     = "Content-Encoding"
	contentEncodingGzip = "gzip"
)

type compressWriter struct {
	gin.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w gin.ResponseWriter) *compressWriter {
	return &compressWriter{
		ResponseWriter: w,
		zw:             gzip.NewWriter(w),
	}
}

func (c *compressWriter) Write(p []byte) (int, error) {
	if c.Status() == http.StatusOK {
		n, err := c.zw.Write(p)
		if err != nil {
			return 0, fmt.Errorf("error writing gzipped bytes: %w", err)
		}
		c.Header().Set("Content-Length", strconv.Itoa(n))

		return n, nil
	} else {
		n, err := c.ResponseWriter.Write(p)
		if err != nil {
			return 0, fmt.Errorf("error writing bytes: %w", err)
		}

		return n, nil
	}
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if c.Status() == http.StatusOK {
		c.Header().Set(contentEncoding, contentEncodingGzip)
	}
	c.ResponseWriter.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *compressWriter) Close() error {
	if err := c.zw.Close(); err != nil {
		return fmt.Errorf("error closing writer: %w", err)
	}
	return nil
}

type compressReader struct {
	io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("error crating gzip reader: %w", err)
	}

	return &compressReader{
		ReadCloser: r,
		zr:         zr,
	}, nil
}

func (c compressReader) Read(p []byte) (int, error) {
	n, err := c.zr.Read(p)
	if err != nil {
		return 0, fmt.Errorf("error reading gzipped: %w", err)
	}
	return n, nil
}

func (c *compressReader) Close() error {
	if err := c.zr.Close(); err != nil {
		return fmt.Errorf("error closing reader: %w", err)
	}
	return nil
}

func Compress(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ow := c.Writer

		acceptEncoding := c.Request.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, contentEncodingGzip)
		if supportsGzip {
			// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
			cw := newCompressWriter(c.Writer)
			// меняем оригинальный http.ResponseWriter на новый
			ow = cw
			// не забываем отправить клиенту все сжатые данные после завершения middleware
			defer func() {
				if err := cw.Close(); err != nil {
					logger.Errorf("error closing compress writer: %w", err)
					return
				}
			}()
		}

		contentEncoding := c.Request.Header.Get(contentEncoding)
		sendsGzip := strings.Contains(contentEncoding, contentEncodingGzip)
		if sendsGzip {
			// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
			cr, err := newCompressReader(c.Request.Body)
			if err != nil {
				logger.Errorf("Error compressing: %v", err)
				c.Writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			// меняем тело запроса на новое
			c.Request.Body = cr
			defer func() {
				if err := cr.Close(); err != nil {
					logger.Errorf("error closing compress reader: %w", err)
					return
				}
			}()
		}

		// передаём управление хендлеру
		c.Writer = ow
		c.Next()
	}
}
