import { useId, useState, type ReactNode } from "react"
import { CameraIcon, Trash2Icon } from "lucide-react"

import { buttonVariants, Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { blobToDataUrl, ImageError, resizeImage } from "@/lib/image"
import { cn } from "@/lib/utils"

export interface PickedPhoto {
  /** A foto já reduzida (JPEG), pronta para enviar. */
  blob: Blob
  /** A mesma foto como texto "data:...", para mostrar na tela. */
  previewUrl: string
}

interface Props {
  shape: "circle" | "rect"
  /** A foto que aparece agora (a atual ou a recém-escolhida); null mostra o `placeholder`. */
  previewUrl: string | null
  /** O que aparece quando não há foto (iniciais, um ícone...). */
  placeholder: ReactNode
  label: string
  onPick: (photo: PickedPhoto) => void
  /** Se informado, aparece o botão "Remover" quando há foto. */
  onRemove?: () => void
  /** Enquanto uma foto está sendo enviada: trava os botões. */
  busy?: boolean
  /** Tamanho máximo do maior lado, em pixels (a foto é reduzida no aparelho antes de enviar). */
  maxSide: number
  square?: boolean
}

/**
 * Escolher uma foto: abre a câmera ou a galeria (no celular) ou o seletor de arquivos (no computador),
 * reduz a imagem no aparelho e entrega pronta. Quem usa decide quando enviar (agora ou ao salvar o formulário).
 */
export function PhotoPicker({ shape, previewUrl, placeholder, label, onPick, onRemove, busy, maxSide, square }: Props) {
  const inputId = useId()
  const [error, setError] = useState<string | null>(null)
  const [preparing, setPreparing] = useState(false)

  async function choose(file: File | undefined) {
    if (!file) return

    setError(null)
    setPreparing(true)

    try {
      const blob = await resizeImage(file, { maxSide, square })

      onPick({ blob, previewUrl: await blobToDataUrl(blob) })
    } catch (caught) {
      setError(caught instanceof ImageError ? caught.message : "Não foi possível preparar a foto.")
    } finally {
      setPreparing(false)
    }
  }

  const working = busy || preparing

  return (
    <div className="flex flex-col items-center gap-3">
      <div
        className={cn(
          "relative flex shrink-0 items-center justify-center overflow-hidden border bg-secondary text-primary",
          shape === "circle" ? "size-28 rounded-full" : "aspect-video w-full max-w-sm rounded-xl",
        )}
      >
        {previewUrl ? <img src={previewUrl} alt={label} className="size-full object-cover" /> : placeholder}
        {working && (
          <div className="absolute inset-0 flex items-center justify-center bg-background/60" aria-live="polite">
            <Spinner className="size-6" />
          </div>
        )}
      </div>

      <div className="flex flex-wrap items-center justify-center gap-2">
        {/* o <input> fica escondido dentro do rótulo: tocar no "botão" abre a câmera ou a galeria */}
        <label
          className={cn(
            buttonVariants({ variant: "outline" }),
            "cursor-pointer has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/50",
            working && "pointer-events-none opacity-50",
          )}
        >
          <CameraIcon data-icon="inline-start" />
          {previewUrl ? "Trocar foto" : "Escolher foto"}
          <input
            id={inputId}
            type="file"
            accept="image/*"
            className="sr-only"
            disabled={working}
            onChange={(event) => {
              void choose(event.target.files?.[0])
              event.target.value = "" // deixa escolher a mesma foto de novo
            }}
          />
        </label>

        {onRemove && previewUrl && (
          <Button type="button" variant="ghost" disabled={working} onClick={onRemove}>
            <Trash2Icon data-icon="inline-start" />
            Remover
          </Button>
        )}
      </div>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
    </div>
  )
}
