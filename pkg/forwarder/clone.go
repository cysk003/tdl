package forwarder

import (
	"context"
	"io"

	"github.com/go-faster/errors"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"

	"github.com/iyear/tdl/pkg/dcpool"
	"github.com/iyear/tdl/pkg/tmedia"
)

type CloneOptions struct {
	Pool     dcpool.Pool
	Media    *tmedia.Media
	PartSize int
	Progress uploader.Progress
}

func CloneMedia(ctx context.Context, opts CloneOptions) (tg.InputFileClass, error) {
	r, w := io.Pipe()

	wg, errctx := errgroup.WithContext(ctx)

	wg.Go(func() (rerr error) {
		defer multierr.AppendInvoke(&rerr, multierr.Close(w))

		_, err := downloader.NewDownloader().
			WithPartSize(opts.PartSize).
			Download(opts.Pool.Client(ctx, opts.Media.DC), opts.Media.InputFileLoc).
			Stream(errctx, w)
		if err != nil {
			return errors.Wrap(err, "download")
		}
		return nil
	})

	var file tg.InputFileClass
	wg.Go(func() (rerr error) {
		defer multierr.AppendInvoke(&rerr, multierr.Close(r))

		var err error
		upload := uploader.NewUpload(opts.Media.Name, r, opts.Media.Size)
		file, err = uploader.NewUploader(opts.Pool.Default(ctx)).
			WithPartSize(opts.PartSize).
			WithProgress(opts.Progress).
			Upload(errctx, upload)
		if err != nil {
			return errors.Wrap(err, "upload")
		}
		return nil
	})

	err := wg.Wait()
	if err != nil {
		return nil, errors.Wrap(err, "wait")
	}

	return file, nil
}