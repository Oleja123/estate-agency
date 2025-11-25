package property

import imagepkg "github.com/Oleja123/estate-agency/internal/domain/image"

// PropertyImage is an alias to the image.PropertyImage in the new package.
// This keeps backward compatibility for code still importing
// internal/domain/property while moving the canonical definition to
// internal/domain/image.
type PropertyImage = imagepkg.PropertyImage

// (ImageRepository alias moved to image_storage.go for clarity.)
