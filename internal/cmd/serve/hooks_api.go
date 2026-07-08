// hooks_api.go registers the read-only endpoint the frontend uses to list
// available post-sync hook scripts, so the hook picker only ever offers
// names that actually resolve to something runnable.
package serve

import (
	"net/http"

	"github.com/asano69/hatchards/internal/hook"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// RegisterHooksAPI registers GET /api/hooks, which lists every hook name
// available under hooksDir. A missing hooksDir yields an empty list rather
// than an error (see hook.List), so installations that never configured a
// hooks directory see an empty picker instead of a broken page.
func RegisterHooksAPI(r *router.Router[*core.RequestEvent], hooksDir string) {
	r.GET("/api/hooks", func(e *core.RequestEvent) error {
		names, err := hook.List(hooksDir)
		if err != nil {
			return e.BadRequestError("list hooks failed", err)
		}
		return e.JSON(http.StatusOK, map[string]any{"hooks": names})
	}).Bind(apis.RequireSuperuserAuth())
}
