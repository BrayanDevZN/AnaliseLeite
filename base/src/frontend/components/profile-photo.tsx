import { initials } from "@/lib/format"
import { useUserPhoto } from "@/lib/photos"
import type { User } from "@/lib/types"
import { cn } from "@/lib/utils"

/** A foto de perfil em um círculo grande, ou as iniciais se a pessoa não tem foto. O tamanho vem do className. */
export function ProfilePhoto({ user, className }: { user: Pick<User, "id" | "name" | "has_profile_photo">; className?: string }) {
  const photo = useUserPhoto(user)

  return (
    <div className={cn("flex shrink-0 items-center justify-center overflow-hidden rounded-full border-2 border-primary/40 bg-secondary text-primary", className)}>
      {photo ? (
        <img src={photo} alt={`Foto de ${user.name}`} className="size-full object-cover" />
      ) : (
        <span className="text-2xl font-semibold" aria-hidden="true">
          {initials(user.name)}
        </span>
      )}
    </div>
  )
}
