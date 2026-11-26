package schema

import "net/http"

// MaxPhotoSize e o maior tamanho de uma foto (em bytes). O frontend reduz a foto antes de enviar.
const MaxPhotoSize = 2 << 20

// photoTypes sao os unicos tipos aceitos. Uma foto servida pela API e conteudo enviado por um usuario: se o
// tipo nao fosse conferido, alguem poderia guardar uma pagina HTML como "foto" e ela rodaria no endereco da API.
var photoTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// ValidatePhoto confere o tamanho e o tipo da imagem (pelos primeiros bytes, nao pelo nome do arquivo) e devolve
// o tipo (Content-Type) para servir a foto. O erro e um *ValidationError no campo "photo".
func ValidatePhoto(data []byte) (contentType string, err error) {
	var v validator

	switch {
	case len(data) == 0:
		v.add("photo", "is empty")

	case len(data) > MaxPhotoSize:
		v.add("photo", "photo is too big")

	default:
		detected := http.DetectContentType(data)

		if !photoTypes[detected] {
			v.add("photo", "must be a jpeg, png, webp or gif image")
		} else {
			return detected, nil
		}
	}

	return "", v.err()
}

// PhotoRoutes sao as rotas de foto de perfil e de foto de fazenda. O corpo do envio e a imagem em si
// (Content-Type image/jpeg, image/png, image/webp ou image/gif), e a resposta de GET e a imagem.
var PhotoRoutes = []Route{
	{Method: http.MethodGet, Path: "/users/{id}/photo", Description: "foto de perfil (so o proprio usuario ou um admin)"},
	{Method: http.MethodPut, Path: "/me/photo", Description: "troca a propria foto de perfil", Response: UserResponse{}},
	{Method: http.MethodDelete, Path: "/me/photo", Description: "remove a propria foto de perfil", Response: UserResponse{}},
	{Method: http.MethodGet, Path: "/farms/{id}/photo", Description: "foto da fazenda"},
	{Method: http.MethodPut, Path: "/farms/{id}/photo", Description: "troca a foto da fazenda", Response: FarmResponse{}},
	{Method: http.MethodDelete, Path: "/farms/{id}/photo", Description: "remove a foto da fazenda", Response: FarmResponse{}},
}
