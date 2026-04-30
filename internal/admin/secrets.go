package admin

import (
	"errors"
	"net/http"
	"sort"

	"codeberg.org/jhot/hats/internal/admin/templates"
	"codeberg.org/jhot/hats/internal/secrets"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func secretListHandler(w http.ResponseWriter, r *http.Request) {
	ss, err := adminSecretStore.List()
	if err != nil {
		http.Error(w, "failed to list secrets", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.Secrets(secretDataList(ss)).Render(r.Context(), w)
}

func secretTableHandler(w http.ResponseWriter, r *http.Request) {
	ss, err := adminSecretStore.List()
	if err != nil {
		http.Error(w, "failed to list secrets", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.SecretList(secretDataList(ss)).Render(r.Context(), w)
}

func secretNewHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.SecretCreateForm("", "basic_string").Render(r.Context(), w)
}

func secretEditHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	s, err := adminSecretStore.Get(name)
	if err != nil {
		http.Error(w, "secret not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.SecretEditForm(templates.SecretData{
		Name:      s.Name,
		Type:      string(s.Type),
		CreatedAt: s.CreatedAt.Format("2006-01-02 15:04"),
		UpdatedAt: s.UpdatedAt.Format("2006-01-02 15:04"),
	}, s.Value, "").Render(r.Context(), w)
}

func secretCreateAPIHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	typStr := r.FormValue("type")
	value := r.FormValue("value")

	if name == "" || typStr == "" || value == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SecretCreateForm("name, type, and value are required", typStr).Render(r.Context(), w)
		return
	}

	typ := secrets.SecretType(typStr)
	switch typ {
	case secrets.TypeBasicString, secrets.TypeEncryptedString, secrets.TypeYAML, secrets.TypeJSON:
	default:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SecretCreateForm("invalid type", typStr).Render(r.Context(), w)
		return
	}

	if err := adminSecretStore.Create(name, value, typ); err != nil {
		msg := "failed to create secret: " + err.Error()
		if errors.Is(err, secrets.ErrInvalidName) {
			msg = "name must contain only letters, numbers, hyphens, and underscores"
		} else if errors.Is(err, secrets.ErrNameConflict) {
			msg = "a secret with that name already exists"
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.SecretCreateForm(msg, typStr).Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.SecretSaved("created").Render(r.Context(), w)
}

func secretUpdateAPIHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	value := r.FormValue("value")
	if value == "" {
		http.Error(w, "value is required", http.StatusBadRequest)
		return
	}
	if err := adminSecretStore.Update(name, value); err != nil {
		http.Error(w, "failed to update secret: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.SecretSaved("updated").Render(r.Context(), w)
}

func secretDeleteAPIHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := adminSecretStore.Delete(name); err != nil {
		http.Error(w, "failed to delete secret: "+err.Error(), http.StatusInternalServerError)
		return
	}
	ss, err := adminSecretStore.List()
	if err != nil {
		http.Error(w, "failed to list secrets", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.SecretList(secretDataList(ss)).Render(r.Context(), w)
}

func secretJSONListHandler(w http.ResponseWriter, r *http.Request) {
	ss, err := adminSecretStore.List()
	if err != nil {
		http.Error(w, "failed to list secrets", http.StatusInternalServerError)
		return
	}
	render.JSON(w, r, secretDataList(ss))
}

func secretDataList(metas []*secrets.SecretMeta) []templates.SecretData {
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].Name < metas[j].Name
	})
	data := make([]templates.SecretData, len(metas))
	for i, m := range metas {
		data[i] = templates.SecretData{
			Name:      m.Name,
			Type:      string(m.Type),
			CreatedAt: m.CreatedAt.Format("2006-01-02 15:04"),
			UpdatedAt: m.UpdatedAt.Format("2006-01-02 15:04"),
		}
	}
	return data
}
