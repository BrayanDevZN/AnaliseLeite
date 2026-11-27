package handles

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"

	"github.com/BrayanDevZN/AnaliseLeite/base/src/backend/schema"
)

// readPhoto le o corpo da requisicao (a imagem em si) e confere o tamanho e o tipo.
// Devolve os bytes; o erro e sempre um *schema.ValidationError (422), menos se a leitura da rede falhar.
func readPhoto(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	// um byte a mais que o maximo, para saber que passou do limite sem ler um corpo enorme
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, schema.MaxPhotoSize+1))

	if err != nil {
		var tooBig *http.MaxBytesError

		if errors.As(err, &tooBig) {
			return nil, bodyError("photo", "photo is too big")
		}

		return nil, bodyError("photo", "could not be read")
	}

	if _, err := schema.ValidatePhoto(data); err != nil {
		return nil, err
	}

	return data, nil
}

// servePhoto responde a imagem. O tipo vem dos bytes (so os tipos de imagem aceitos), e os cabecalhos impedem
// o navegador de tratar a foto como pagina ou de executar qualquer coisa dentro dela.
func servePhoto(w http.ResponseWriter, r *http.Request, data []byte) {
	if len(data) == 0 {
		writeJSON(w, http.StatusNotFound, schema.ErrorResponse{Detail: errNotFound.detail})

		return
	}

	contentType, err := schema.ValidatePhoto(data)

	if err != nil {
		writeJSON(w, http.StatusNotFound, schema.ErrorResponse{Detail: errNotFound.detail})

		return
	}

	sum := sha256.Sum256(data)
	etag := `"` + hex.EncodeToString(sum[:8]) + `"`

	header := w.Header()

	header.Set("Content-Type", contentType)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	header.Set("Content-Disposition", "inline")
	// privada: so o navegador de quem pediu guarda; "no-cache" faz ele perguntar (ETag) antes de reusar
	header.Set("Cache-Control", "private, no-cache")
	header.Set("ETag", etag)

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)

		return
	}

	w.Write(data)
}

// getUserPhoto: GET /users/{id}/photo. So o proprio usuario ou um admin; para os outros, "nao achou".
func (h *Handles) getUserPhoto(w http.ResponseWriter, r *http.Request, p *Principal) {
	id, err := pathID(r)

	if err != nil || !p.CanAccessUser(id) {
		h.fail(w, errNotFound)

		return
	}

	user, err := h.db.Users.GetByID(r.Context(), id)

	if err != nil {
		h.fail(w, err)

		return
	}

	servePhoto(w, r, user.ProfilePhoto)
}

// putMyPhoto: PUT /me/photo. O corpo e a imagem.
func (h *Handles) putMyPhoto(w http.ResponseWriter, r *http.Request, p *Principal) {
	data, err := readPhoto(w, r)

	if err != nil {
		h.fail(w, err)

		return
	}

	h.setMyPhoto(w, r, p, data)
}

// deleteMyPhoto: DELETE /me/photo.
func (h *Handles) deleteMyPhoto(w http.ResponseWriter, r *http.Request, p *Principal) {
	h.setMyPhoto(w, r, p, nil)
}

func (h *Handles) setMyPhoto(w http.ResponseWriter, r *http.Request, p *Principal, photo []byte) {
	user := *p.User

	user.ProfilePhoto = photo

	if err := h.db.Users.Update(r.Context(), &user); err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.UserFromModel(&user))
}

// getFarmPhoto: GET /farms/{id}/photo.
func (h *Handles) getFarmPhoto(w http.ResponseWriter, r *http.Request, p *Principal) {
	farm, err := h.ownedFarm(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	photo, err := h.db.Farms.GetPhoto(r.Context(), farm.ID)

	if err != nil {
		h.fail(w, err)

		return
	}

	servePhoto(w, r, photo)
}

// putFarmPhoto: PUT /farms/{id}/photo. O corpo e a imagem.
func (h *Handles) putFarmPhoto(w http.ResponseWriter, r *http.Request, p *Principal) {
	farm, err := h.ownedFarm(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	data, err := readPhoto(w, r)

	if err != nil {
		h.fail(w, err)

		return
	}

	h.setFarmPhoto(w, r, farm.ID, data)
}

// deleteFarmPhoto: DELETE /farms/{id}/photo.
func (h *Handles) deleteFarmPhoto(w http.ResponseWriter, r *http.Request, p *Principal) {
	farm, err := h.ownedFarm(r, p)

	if err != nil {
		h.fail(w, err)

		return
	}

	h.setFarmPhoto(w, r, farm.ID, nil)
}

func (h *Handles) setFarmPhoto(w http.ResponseWriter, r *http.Request, id int64, photo []byte) {
	if err := h.db.Farms.SetPhoto(r.Context(), id, photo); err != nil {
		h.fail(w, err)

		return
	}

	farm, err := h.db.Farms.GetByID(r.Context(), id)

	if err != nil {
		h.fail(w, err)

		return
	}

	writeJSON(w, http.StatusOK, schema.FarmFromModel(farm))
}
