/** O tamanho máximo, em pixels, do maior lado da foto enviada. */
export const AVATAR_SIDE = 384
export const COVER_SIDE = 1024

export class ImageError extends Error {}

/** Lê um blob e devolve ele como texto "data:..." (serve de src de <img> sem precisar liberar nada depois). */
export function blobToDataUrl(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()

    reader.onload = () => resolve(String(reader.result))
    reader.onerror = () => reject(new ImageError("Não foi possível ler a imagem."))
    reader.readAsDataURL(blob)
  })
}

/**
 * Reduz a foto no próprio aparelho antes de enviar: uma foto de celular tem vários MB, e o servidor aceita até 2 MB.
 * Vira JPEG. Com `square`, corta um quadrado do centro (para o avatar redondo); sem, só diminui.
 * O EXIF (a rotação que o celular grava) é aplicado, então a foto não chega deitada.
 */
export async function resizeImage(file: File, { maxSide, square = false }: { maxSide: number; square?: boolean }): Promise<Blob> {
  if (!file.type.startsWith("image/")) {
    throw new ImageError("Escolha um arquivo de imagem.")
  }

  let bitmap: ImageBitmap

  try {
    bitmap = await createImageBitmap(file, { imageOrientation: "from-image" })
  } catch {
    throw new ImageError("Não foi possível abrir essa imagem. Tente uma foto JPG ou PNG.")
  }

  // quadrado do centro, ou a imagem inteira
  const side = Math.min(bitmap.width, bitmap.height)
  const sx = square ? (bitmap.width - side) / 2 : 0
  const sy = square ? (bitmap.height - side) / 2 : 0
  const sw = square ? side : bitmap.width
  const sh = square ? side : bitmap.height

  // nunca aumenta a foto, so diminui
  const scale = Math.min(1, maxSide / Math.max(sw, sh))
  const width = Math.max(1, Math.round(sw * scale))
  const height = Math.max(1, Math.round(sh * scale))

  const canvas = document.createElement("canvas")

  canvas.width = width
  canvas.height = height

  const context = canvas.getContext("2d")

  if (!context) {
    throw new ImageError("Seu navegador não conseguiu preparar a foto.")
  }

  // fundo branco: um PNG com transparência viraria preto no JPEG
  context.fillStyle = "#ffffff"
  context.fillRect(0, 0, width, height)
  context.drawImage(bitmap, sx, sy, sw, sh, 0, 0, width, height)
  bitmap.close()

  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, "image/jpeg", 0.86))

  if (!blob) {
    throw new ImageError("Não foi possível preparar a foto.")
  }

  return blob
}
