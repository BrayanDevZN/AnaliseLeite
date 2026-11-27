import { TractorIcon } from "lucide-react"

import { useFarmPhoto } from "@/lib/photos"
import type { Farm } from "@/lib/types"
import { cn } from "@/lib/utils"

/** Miniatura da fazenda: a foto, ou um trator no lugar. O tamanho vem do className (ex.: "size-14"). */
export function FarmThumb({ farm, className }: { farm: Pick<Farm, "id" | "has_photo">; className?: string }) {
  const photo = useFarmPhoto(farm)

  if (photo) {
    return <img src={photo} alt="" className={cn("shrink-0 rounded-lg object-cover", className)} />
  }

  return (
    <div className={cn("flex shrink-0 items-center justify-center rounded-lg bg-secondary text-primary", className)} aria-hidden="true">
      <TractorIcon className="size-1/2" />
    </div>
  )
}
