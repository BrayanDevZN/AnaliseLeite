import type { ComponentProps } from "react"

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { initials } from "@/lib/format"
import { useUserPhoto } from "@/lib/photos"
import type { User } from "@/lib/types"

/** Avatar de um usuário: a foto, ou as iniciais enquanto carrega ou se ele não tem foto. */
export function UserAvatar({ user, size }: { user: Pick<User, "id" | "name" | "has_profile_photo">; size?: ComponentProps<typeof Avatar>["size"] }) {
  const photo = useUserPhoto(user)

  return (
    <Avatar size={size}>
      {photo && <AvatarImage src={photo} alt="" />}
      <AvatarFallback>{initials(user.name)}</AvatarFallback>
    </Avatar>
  )
}
