import { useState, type FormEvent } from "react"
import { LogOutIcon, ShieldCheckIcon } from "lucide-react"
import { useNavigate } from "react-router"

import { ErrorAlert } from "@/components/error-alert"
import { PageHeader } from "@/components/page-header"
import { SubmitButton } from "@/components/submit-button"
import { TextField } from "@/components/text-field"
import { PasswordChecklist } from "@/components/password-checklist"
import { PhotoPicker } from "@/components/photo-picker"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { FieldGroup } from "@/components/ui/field"
import { api, ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { initials } from "@/lib/format"
import { AVATAR_SIDE } from "@/lib/image"
import { useMyPhoto, useUserPhoto } from "@/lib/photos"
import { notify } from "@/lib/notify"
import type { User } from "@/lib/types"
import { GMAIL_MESSAGE, isGmail, passwordProblem } from "@/lib/validation"

export default function ProfilePage() {
  const { user, setUser, logout } = useAuth()
  const navigate = useNavigate()

  const [name, setName] = useState(user?.name ?? "")
  const [email, setEmail] = useState(user?.email ?? "")
  const [password, setPassword] = useState("")
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<unknown>(null)
  const [local, setLocal] = useState<Record<string, string>>({})
  const photo = useUserPhoto(user)
  const myPhoto = useMyPhoto()

  if (!user) return null

  const fields = { ...(error instanceof ApiError ? error.fields : {}), ...local }

  async function save(event: FormEvent) {
    event.preventDefault()

    // a regra do email só vale se ele MUDOU (uma conta antiga com outro domínio continua editável)
    const problems: Record<string, string> = {}

    if (email.trim().toLowerCase() !== user!.email && !isGmail(email)) problems.email = GMAIL_MESSAGE

    if (password) {
      const weak = passwordProblem(password)

      if (weak) problems.password = weak
    }

    setLocal(problems)

    if (Object.keys(problems).length > 0) return

    setPending(true)
    setError(null)

    try {
      // só manda o que mudou
      const changes: Record<string, string> = {}

      if (name.trim() !== user!.name) changes.name = name.trim()
      if (email.trim().toLowerCase() !== user!.email) changes.email = email.trim()
      if (password) changes.password = password

      if (Object.keys(changes).length === 0) {
        notify.success("Nada para salvar", "Você não mudou nenhum dado.")

        return
      }

      setUser(await api.patch<User>("/me", changes))
      setPassword("")
      notify.success("Perfil atualizado")
    } catch (caught) {
      setError(caught)
    } finally {
      setPending(false)
    }
  }

  async function leave() {
    try {
      await logout()
      navigate("/entrar", { replace: true })
    } catch (error) {
      notify.error("Não foi possível sair", error instanceof Error ? error.message : undefined)
    }
  }

  return (
    <>
      <PageHeader title="Seu perfil" />

      <Card>
        <CardContent className="flex flex-col items-center gap-4">
          <PhotoPicker
            shape="circle"
            previewUrl={photo}
            placeholder={<span className="text-3xl font-semibold">{initials(user.name)}</span>}
            label="Sua foto"
            busy={myPhoto.set.isPending || myPhoto.remove.isPending}
            maxSide={AVATAR_SIDE}
            square
            onPick={async ({ blob }) => {
              try {
                await myPhoto.set.mutateAsync(blob)
                notify.success("Foto atualizada")
              } catch (caught) {
                notify.error("Não foi possível salvar a foto", caught instanceof Error ? caught.message : undefined)
              }
            }}
            onRemove={async () => {
              try {
                await myPhoto.remove.mutateAsync()
                notify.success("Foto removida")
              } catch (caught) {
                notify.error("Não foi possível remover a foto", caught instanceof Error ? caught.message : undefined)
              }
            }}
          />
          <div className="flex min-w-0 flex-col items-center gap-1.5 text-center">
            <CardTitle className="max-w-full truncate">{user.name}</CardTitle>
            <CardDescription className="flex flex-wrap items-center justify-center gap-2">
              <span className="truncate">{user.email}</span>
              <Badge variant={user.role === "admin" ? "default" : "secondary"}>
                {user.role === "admin" && <ShieldCheckIcon data-icon="inline-start" />}
                {user.role === "admin" ? "Administrador" : "Usuário"}
              </Badge>
            </CardDescription>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Seus dados</CardTitle>
          <CardDescription>{user.has_password ? "Para trocar a senha, digite a nova abaixo." : "Você entra com o Google. Se quiser, crie uma senha também."}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={save} noValidate>
            <FieldGroup>
              {error != null && Object.keys(fields).length === 0 && <ErrorAlert error={error} title="Não foi possível salvar" />}
              <TextField name="name" label="Nome" autoComplete="name" value={name} onChange={(event) => setName(event.target.value)} error={fields.name} />
              <TextField
                name="email"
                label="Email"
                type="email"
                inputMode="email"
                autoComplete="email"
                autoCapitalize="none"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                error={fields.email}
              />
              <TextField
                name="password"
                label="Nova senha"
                type="password"
                autoComplete="new-password"
                description="Deixe em branco para manter a atual."
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                error={fields.password}
              />
              {password && <PasswordChecklist value={password} />}
              <SubmitButton size="lg" pending={pending} pendingLabel="Salvando">
                Salvar alterações
              </SubmitButton>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>

      <Button variant="outline" size="lg" onClick={leave}>
        <LogOutIcon data-icon="inline-start" />
        Sair da conta
      </Button>
    </>
  )
}
