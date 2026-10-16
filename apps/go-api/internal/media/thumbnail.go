package media

import (
	"context"
	"io"
	"strconv"
)

// RunWebPThumbnail génère la miniature animée WebP de videoPath dans webpPath : extract
// secondes de vidéo à partir de skip, 480 px de large, 12 images par seconde, qualité 75,
// compression maximale, sans son, en tenant la place des miniatures (PoolThumbnail : elle
// n'attend pas un transcodage). La sortie d'erreur de ffmpeg va dans stderr (nil :
// écartée).
func RunWebPThumbnail(ctx context.Context, videoPath, webpPath string, skip, extract float64, stderr io.Writer) error {
	return RunFFmpeg(ctx, FFmpegJob{
		Args:   thumbnailArgs(videoPath, webpPath, skip, extract),
		Stderr: stderr,
		Pool:   PoolThumbnail,
	})
}

// thumbnailArgs construit les arguments ffmpeg d'une miniature WebP animée.
func thumbnailArgs(videoPath, webpPath string, skip, extract float64) []string {
	return []string{"-y",
		"-ss", strconv.FormatFloat(skip, 'f', 2, 64),
		"-t", strconv.FormatFloat(extract, 'f', 2, 64),
		"-i", videoPath,
		"-an",
		"-vf", "fps=12,scale=480:-1:flags=lanczos",
		"-c:v", "libwebp",
		"-loop", "0",
		"-q:v", "75",
		"-compression_level", "6",
		"-preset", "picture",
		webpPath,
	}
}
