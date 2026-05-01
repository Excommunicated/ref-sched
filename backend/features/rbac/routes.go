package rbac

import (
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterRoutes registers all RBAC admin routes
func (h *Handler) RegisterRoutes(
	r *mux.Router,
	requireAuth func(http.HandlerFunc) http.HandlerFunc,
	requirePermission func(string, http.HandlerFunc) http.HandlerFunc,
) {
	r.HandleFunc("/api/admin/users/{id}", requireAuth(requirePermission("can_assign_roles", h.DeleteUser))).Methods("DELETE")
	r.HandleFunc("/api/admin/users/{id}/roles", requireAuth(requirePermission("can_assign_roles", h.AssignRoleToUser))).Methods("POST")
	r.HandleFunc("/api/admin/users/{id}/roles/{roleId}", requireAuth(requirePermission("can_assign_roles", h.RevokeRoleFromUser))).Methods("DELETE")
	r.HandleFunc("/api/admin/users/{id}/roles", requireAuth(requirePermission("can_assign_roles", h.GetUserRoles))).Methods("GET")
	r.HandleFunc("/api/admin/roles", requireAuth(requirePermission("can_assign_roles", h.GetAllRoles))).Methods("GET")
	r.HandleFunc("/api/admin/permissions", requireAuth(requirePermission("can_assign_roles", h.GetAllPermissions))).Methods("GET")
}
