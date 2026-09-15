package stores

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MinioClient *minio.Client
var MinioPublicURL string

// InitMinio initializes the shared client and verifies that MinIO is reachable.
func InitMinio() error {
	_ = godotenv.Load()

	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	bucketName := os.Getenv("MINIO_PRODUCT_BUCKET")
	secureStr := os.Getenv("MINIO_SECURE")
	MinioPublicURL = os.Getenv("MINIO_PUBLIC_URL")
	if endpoint == "" || accessKey == "" || secretKey == "" || bucketName == "" || MinioPublicURL == "" {
		return fmt.Errorf("MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, MINIO_PRODUCT_BUCKET, and MINIO_PUBLIC_URL must be configured")
	}

	secure, err := strconv.ParseBool(secureStr)
	if err != nil {
		return fmt.Errorf("invalid MINIO_SECURE value %q: %w", secureStr, err)
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return fmt.Errorf("create MinIO client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := client.BucketExists(ctx, bucketName); err != nil {
		return fmt.Errorf("connect to MinIO endpoint %q: %w", endpoint, err)
	}

	MinioClient = client
	log.Println("✅ MinIO terhubung ke:", endpoint)
	return nil
}

// UploadToMinio: upload ke bucket tertentu
func UploadToMinio(file multipart.File, bucketName string, folderName string, filename string, contentType string, size int64) (string, error) {
	if MinioClient == nil {
		return "", fmt.Errorf("MinIO client belum diinisialisasi")
	}
	if bucketName == "" {
		return "", fmt.Errorf("nama bucket MinIO kosong")
	}

	ctx := context.Background()

	// Pastikan bucket ada
	exists, err := MinioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return "", fmt.Errorf("cek bucket gagal: %w", err)
	}
	if !exists {
		log.Println("Bucket belum ada, membuat:", bucketName)
		if err := MinioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			return "", fmt.Errorf("❌ gagal membuat bucket: %w", err)
		}
		log.Println("✅ Berhasil membuat bucket:", bucketName)
	}

	// --- [1] Sanitasi nama institusi untuk folder ---
	subFolder := "unknown"
	if folderName != "" {
		// Hapus spasi, simbol aneh, dan huruf besar → lowercase + dash
		subFolder = strings.ToLower(strings.ReplaceAll(folderName, " ", "-"))
		subFolder = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				return r
			}
			return -1
		}, subFolder)
	}

	// --- [2] Siapkan nama file ---
	objectName := fmt.Sprintf("%s/%s-%s", subFolder, uuid.NewString(), filename)

	// --- [3] Cek apakah file gambar ---
	ext := strings.ToLower(filepath.Ext(filename))
	isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp"

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("gagal membaca file: %w", err)
	}

	var uploadReader *bytes.Reader
	var uploadSize int64
	var uploadContentType string

	if isImage {
		img, _, err := image.Decode(bytes.NewReader(fileBytes))
		if err != nil {
			return "", fmt.Errorf("gagal decode image: %w", err)
		}

		resized := imaging.Resize(img, 1280, 0, imaging.Lanczos)

		buf := new(bytes.Buffer)
		switch ext {
		case ".png":
			err = png.Encode(buf, resized)
			uploadContentType = "image/png"
		default:
			err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: 70})
			uploadContentType = "image/jpeg"
		}
		if err != nil {
			return "", fmt.Errorf("gagal kompres image: %w", err)
		}

		uploadReader = bytes.NewReader(buf.Bytes())
		uploadSize = int64(buf.Len())
	} else {
		uploadReader = bytes.NewReader(fileBytes)
		uploadSize = int64(len(fileBytes))
		uploadContentType = contentType
	}

	// --- [4] Upload ke MinIO ---
	_, err = MinioClient.PutObject(
		ctx,
		bucketName,
		objectName,
		uploadReader,
		uploadSize,
		minio.PutObjectOptions{ContentType: uploadContentType},
	)
	if err != nil {
		return "", fmt.Errorf("upload ke MinIO gagal: %w", err)
	}

	// --- [5] Return URL publik ---
	return fmt.Sprintf("%s/%s/%s", MinioPublicURL, bucketName, objectName), nil
}

func DeleteMinioFiles(bucketName string, fileURL string) {
	ctx := context.Background()

	if MinioClient == nil {
		log.Println("⚠️ MinioClient belum diinisialisasi.")
		return
	}

	if bucketName == "" {
		log.Println("⚠️ Bucket name kosong, lewati penghapusan file.")
		return
	}

	if fileURL == "" {
		log.Println("⚠️ File URL kosong, lewati penghapusan file.")
		return
	}

	publicPrefix := fmt.Sprintf("%s/%s/", strings.TrimSuffix(MinioPublicURL, "/"), bucketName)
	objectName := strings.TrimPrefix(fileURL, publicPrefix)

	if objectName == "" || objectName == fileURL {
		log.Printf("⚠️ Tidak bisa ekstrak object dari %s", fileURL)
		return
	}

	err := MinioClient.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		log.Printf("❌ Gagal hapus object %s: %v", objectName, err)
	} else {
		log.Printf("🗑️ Hapus file: %s", objectName)
	}
}
