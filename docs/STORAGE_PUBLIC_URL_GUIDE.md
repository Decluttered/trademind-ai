# Storage Public URL Configuration Guide (H1.3)

> Doudian image uploads and platform-listing external image links both require an image URL that is accessible over **public HTTPS**.

## Configuration Fields

| Setting key | Description |
| --- | --- |
| `storage.public_base` | Generic public prefix (recommended) |
| `s3_public_base` / `cos_public_base` / `oss_public_base` | Cloud storage-specific prefixes |

Backend resolution: `backend/internal/pkg/storagepublic/public_base.go` → `ResolvePublicBase()`

## Local Storage Notes

- The default `local` + `/static` configuration **cannot** pass the Doudian image-upload preflight check
- An HTTPS domain accessible from the public internet must be configured, e.g. `https://cdn.example.com/uploads`

## Configuration Steps

1. Open **Settings → Storage**
2. Fill in `public_base` (a full HTTPS prefix, with no trailing-slash conflicts)
3. Click **Test Public Access** (uploads a probe image → verifies with an anonymous GET)
4. Confirm "Storage Public Access" shows as configured under **Settings → Config Status**

## Missing-Configuration Message (standard copy)

> Storage does not yet have a public access address configured. Before uploading Doudian images, ensure product images are accessible via a public URL.

## Relationship to AI Images

- After a processed result is written to Storage, if `public_base` is missing, the failure code is: `storage_public_url_missing`
- Failed-task center category: `ai_image_storage_public_url_missing`

## Related Entry Points

- `/settings/storage`
- `/settings/config-status` (`storage_public_access` card)
- `/settings/platforms` → Doudian production preflight → `storage.public_access`

## Security

- Tests do not leak internal disk paths
- Logs do not output full secret values
