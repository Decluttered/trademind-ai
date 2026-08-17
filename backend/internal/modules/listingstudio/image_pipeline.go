package listingstudio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

const maxImageBytes int64 = 12 << 20

type ImageStorage interface {
	Save(context.Context, int64, string, []byte, string) (string, error)
}
type BackgroundRemover interface {
	RemoveBackground(context.Context, []byte, string) ([]byte, string, error)
}
type ImagePipeline struct {
	DB      *gorm.DB
	Client  *http.Client
	Storage ImageStorage
	Remover BackgroundRemover
}

func PublicImageURL(ctx context.Context, raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return nil, fmt.Errorf("image URL must be public HTTPS")
	}
	if strings.EqualFold(u.Hostname(), "localhost") {
		return nil, fmt.Errorf("private image host is forbidden")
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", u.Hostname())
	if err != nil {
		return nil, fmt.Errorf("resolve image host: %w", err)
	}
	for _, addr := range addresses {
		if !isPublicAddress(addr) {
			return nil, fmt.Errorf("private image address is forbidden")
		}
	}
	return u, nil
}

func isPublicAddress(addr netip.Addr) bool {
	return addr.IsValid() && !addr.IsPrivate() && !addr.IsLoopback() && !addr.IsLinkLocalUnicast() && !addr.IsUnspecified() && !addr.IsMulticast()
}

func publicImageClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid image host: %w", err)
		}
		addresses, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("resolve image host: %w", err)
		}
		for _, addr := range addresses {
			netipAddr, ok := netip.AddrFromSlice(addr)
			if !ok || !isPublicAddress(netipAddr) {
				return nil, fmt.Errorf("private image address is forbidden")
			}
		}
		if len(addresses) == 0 {
			return nil, fmt.Errorf("image host has no addresses")
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
	}}
	return &http.Client{Transport: transport, Timeout: 20 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
}

func stripMetadata(data []byte, contentType string) ([]byte, int, int, string, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, "", fmt.Errorf("image decode: %w", err)
	}
	bounds := img.Bounds()
	var out bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&out, img, &jpeg.Options{Quality: 92})
		contentType = "image/jpeg"
	case "png":
		err = png.Encode(&out, img)
		contentType = "image/png"
	default:
		return nil, 0, 0, "", fmt.Errorf("unsupported image format %s", format)
	}
	if err != nil {
		return nil, 0, 0, "", err
	}
	return out.Bytes(), bounds.Dx(), bounds.Dy(), contentType, nil
}

func (p *ImagePipeline) Ingest(ctx context.Context, w int64, origin string) (*ImageAsset, error) {
	if p == nil || p.DB == nil || p.Storage == nil {
		return nil, fmt.Errorf("image pipeline unavailable")
	}
	u, err := PublicImageURL(ctx, origin)
	if err != nil {
		return nil, err
	}
	client := p.Client
	if client == nil {
		client = publicImageClient()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "image/jpeg,image/png")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image download returned %d", res.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxImageBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxImageBytes {
		return nil, fmt.Errorf("image exceeds size limit")
	}
	ct := strings.ToLower(strings.TrimSpace(strings.Split(res.Header.Get("Content-Type"), ";")[0]))
	if ct != "image/jpeg" && ct != "image/png" {
		return nil, fmt.Errorf("unsupported image content type")
	}
	clean, width, height, ct, err := stripMetadata(raw, ct)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(clean)
	hash := hex.EncodeToString(sum[:])
	ext := filepath.Ext(u.Path)
	if ct == "image/jpeg" {
		ext = ".jpg"
	} else {
		ext = ".png"
	}
	key := fmt.Sprintf("mindbay/t%d/images/%s%s", w, hash, ext)
	stored, err := p.Storage.Save(ctx, w, key, clean, ct)
	if err != nil {
		return nil, err
	}
	row := ImageAsset{WorkspaceID: w, OriginURL: u.String(), SHA256: hash, StorageKey: stored, ContentType: ct, SizeBytes: int64(len(clean)), Width: width, Height: height, Flags: j(map[string]any{"metadataStripped": true, "backgroundRemoval": "disabled"})}
	if err := p.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	derivative := ImageDerivative{WorkspaceID: w, ImageAssetID: row.ID, Kind: "metadata_stripped", StorageKey: stored, SHA256: hash, ContentType: ct}
	if err := p.DB.WithContext(ctx).Create(&derivative).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
