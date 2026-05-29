package handler

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
)

func requireAuth(w http.ResponseWriter, r *http.Request, svcCtx *svc.ServiceContext) (auth.BearerContext, bool) {
	ctx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
	if ctx.UserID <= 0 {
		httpresp.Write(w, http.StatusUnauthorized, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusUnauthorized), "请先登录"), nil)
		return ctx, false
	}
	return ctx, true
}

func pathSegments(r *http.Request) []string {
	return strings.Split(strings.Trim(r.URL.Path, "/"), "/")
}

func parseIDBeforeSuffix(r *http.Request, suffix string) (int64, error) {
	parts := pathSegments(r)
	if len(parts) < 2 || parts[len(parts)-1] != suffix {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(parts[len(parts)-2], 10, 64)
}

func parseArtistIDFromPath(parts []string, suffix string) (int64, error) {
	for i, p := range parts {
		if p == "artists" && i+2 < len(parts) && parts[i+2] == suffix {
			return strconv.ParseInt(parts[i+1], 10, 64)
		}
	}
	return 0, strconv.ErrSyntax
}

func parsePlaylistID(r *http.Request) (int64, error) {
	parts := pathSegments(r)
	for i, p := range parts {
		if p == "playlists" && i+1 < len(parts) {
			return strconv.ParseInt(parts[i+1], 10, 64)
		}
	}
	return 0, strconv.ErrSyntax
}

func parsePlaylistAndSongID(r *http.Request) (playlistID, songID int64, err error) {
	parts := pathSegments(r)
	for i, p := range parts {
		if p == "playlists" && i+4 < len(parts) && parts[i+2] == "songs" {
			playlistID, err = strconv.ParseInt(parts[i+1], 10, 64)
			if err != nil {
				return 0, 0, err
			}
			songID, err = strconv.ParseInt(parts[i+3], 10, 64)
			return playlistID, songID, err
		}
	}
	return 0, 0, strconv.ErrSyntax
}

func getLocalFilePath(svcCtx *svc.ServiceContext, assetURL string) string {
	if strings.HasPrefix(assetURL, "http://") || strings.HasPrefix(assetURL, "https://") {
		cdnBase := strings.TrimRight(svcCtx.Config.Storage.CdnBaseUrl, "/")
		if cdnBase != "" && strings.HasPrefix(assetURL, cdnBase) {
			objectKey := strings.TrimPrefix(assetURL, cdnBase)
			objectKey = strings.TrimLeft(objectKey, "/")
			return filepath.Join(svcCtx.Config.Local.Root, filepath.FromSlash(objectKey))
		}
		return ""
	}
	if strings.HasPrefix(assetURL, "/") {
		return filepath.Join(svcCtx.Config.Local.Root, filepath.FromSlash(strings.TrimLeft(assetURL, "/")))
	}
	return filepath.Join(svcCtx.Config.Local.Root, filepath.FromSlash(assetURL))
}
