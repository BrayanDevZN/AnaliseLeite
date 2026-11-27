import { useState } from "react"
import { useNavigate } from "react-router"

import { AuthLayout } from "@/components/auth-layout"
import { ErrorAlert } from "@/components/error-alert"
import { PhotoPicker, type PickedPhoto } from "@/components/photo-picker"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import { useAuth } from "@/lib/auth"
import { initials } from "@/lib/format"
import { AVATAR_SIDE } from "@/lib/image"
import { notify } from "@/lib/notify"
import { useMyPhoto } from "@/lib/photos"

/** Segundo passo do cadastro: a pessoa pode pôr uma foto agora ou pular (dá para trocar depois, no Perfil). */
export default function PhotoPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const { set } = useMyPhoto()
  const [picked, setPicked] = useState<PickedPhoto | null>(null)

  if (!user) return null

  const next = () => navigate("/", { replace: true })

  async function save() {
    if (!picked) return

    try {
      await set.mutateAsync(picked.blob)
      notify.success("Foto salva")
      next()
    } catch {
      // o erro aparece na tela (set.error)
    }
  }

  return (
    <AuthLayout mascot={false} title={`Bem-vindo, ${user.name.split(" ")[0]}!`} description="Quer adicionar uma foto? Ela aparece no seu perfil. Dá para trocar depois.">
      <Card className="w-full">
        <CardContent className="flex flex-col gap-4">
          <PhotoPicker
            shape="circle"
            previewUrl={picked?.previewUrl ?? null}
            placeholder={<span className="text-3xl font-semibold">{initials(user.name)}</span>}
            label="Sua foto"
            onPick={setPicked}
            busy={set.isPending}
            maxSide={AVATAR_SIDE}
            square
          />

          {set.error && <ErrorAlert error={set.error} title="Não foi possível salvar a foto" />}

          <Button size="lg" disabled={!picked || set.isPending} onClick={save}>
            {set.isPending && <Spinner data-icon="inline-start" />}
            {set.isPending ? "Salvando" : "Salvar foto e continuar"}
          </Button>
          <Button variant="ghost" size="lg" disabled={set.isPending} onClick={next}>
            Pular por enquanto
          </Button>
        </CardContent>
      </Card>
    </AuthLayout>
  )
}
