import { useState, type FormEvent } from "react"
import { PencilIcon, PlusIcon, ShieldCheckIcon, Trash2Icon } from "lucide-react"

import { ConfirmDelete } from "@/components/confirm-delete"
import { ErrorAlert } from "@/components/error-alert"
import { FormDrawer } from "@/components/form-drawer"
import { ListSkeleton } from "@/components/list-skeleton"
import { PageHeader } from "@/components/page-header"
import { SubmitButton } from "@/components/submit-button"
import { TextField } from "@/components/text-field"
import { UserAvatar } from "@/components/user-avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { notify } from "@/lib/notify"
import { useCreateUser, useDeleteUser, useUpdateUser, useUsers } from "@/lib/queries"
import type { User } from "@/lib/types"

const ROLES = [
  { value: "user", label: "Usuário" },
  { value: "admin", label: "Administrador" },
]

function UserFormDrawer({ open, onOpenChange, user }: { open: boolean; onOpenChange: (open: boolean) => void; user?: User }) {
  return (
    <FormDrawer open={open} onOpenChange={onOpenChange} title={user ? "Editar usuário" : "Novo usuário"}>
      {/* o formulário só existe com a folha aberta: nasce novo a cada abertura */}
      <UserForm user={user} onDone={() => onOpenChange(false)} />
    </FormDrawer>
  )
}

function UserForm({ user, onDone }: { user?: User; onDone: () => void }) {
  const create = useCreateUser()
  const update = useUpdateUser(user?.id ?? 0)
  const mutation = user ? update : create

  const [name, setName] = useState(user?.name ?? "")
  const [email, setEmail] = useState(user?.email ?? "")
  const [password, setPassword] = useState("")
  const [role, setRole] = useState<string>(user?.role ?? "user")

  const fields = mutation.error instanceof ApiError ? mutation.error.fields : {}

  async function submit(event: FormEvent) {
    event.preventDefault()

    try {
      await mutation.mutateAsync({ name: name.trim(), email: email.trim(), role, ...(password ? { password } : {}) })
      notify.success(user ? "Usuário atualizado" : "Usuário criado")
      onDone()
    } catch {
      // o erro aparece no formulário
    }
  }

  return (
    <form onSubmit={submit} noValidate>
      <FieldGroup>
        {mutation.error && Object.keys(fields).length === 0 && <ErrorAlert error={mutation.error} />}
        <TextField name="name" label="Nome" autoComplete="off" value={name} onChange={(event) => setName(event.target.value)} error={fields.name} required />
        <TextField
          name="email"
          label="Email"
          type="email"
          inputMode="email"
          autoCapitalize="none"
          autoComplete="off"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          error={fields.email}
          required
        />
        <TextField
          name="password"
          label={user ? "Nova senha" : "Senha"}
          type="password"
          autoComplete="new-password"
          description={user ? "Deixe em branco para manter a atual." : "Opcional: sem senha, a pessoa entra só pelo Google."}
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          error={fields.password}
        />
        <Field>
          <FieldLabel htmlFor="role">Papel</FieldLabel>
          <Select value={role} onValueChange={(value) => setRole(value ?? "user")} items={ROLES}>
            <SelectTrigger id="role" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {ROLES.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </Field>
        <SubmitButton size="lg" pending={mutation.isPending} pendingLabel="Salvando" disabled={!name.trim() || !email.trim()}>
          Salvar
        </SubmitButton>
      </FieldGroup>
    </form>
  )
}

export default function UsersPage() {
  const { user: me } = useAuth()
  const users = useUsers()
  const remove = useDeleteUser()
  const [drawer, setDrawer] = useState<{ open: boolean; user?: User }>({ open: false })

  async function apagar(user: User) {
    try {
      await remove.mutateAsync(user.id)
      notify.success("Usuário apagado")
    } catch (error) {
      notify.error("Não foi possível apagar", error instanceof Error ? error.message : undefined)
    }
  }

  return (
    <>
      <PageHeader
        title="Usuários"
        description="Quem pode entrar no sistema."
        actions={
          <Button onClick={() => setDrawer({ open: true })}>
            <PlusIcon data-icon="inline-start" />
            Novo usuário
          </Button>
        }
      />

      {users.isPending && <ListSkeleton />}
      {users.isError && <ErrorAlert error={users.error} title="Não foi possível carregar os usuários" />}

      <ul className="flex flex-col gap-3">
        {users.data?.map((item) => (
          <li key={item.id}>
            <Card size="sm">
              <CardHeader className="flex-row items-center gap-3">
                <UserAvatar user={item} />
                <div className="flex min-w-0 flex-1 flex-col gap-1">
                  <CardTitle className="truncate">{item.name}</CardTitle>
                  <CardDescription className="flex flex-wrap items-center gap-2">
                    <span className="truncate">{item.email}</span>
                    <Badge variant={item.role === "admin" ? "default" : "secondary"}>
                      {item.role === "admin" && <ShieldCheckIcon data-icon="inline-start" />}
                      {item.role === "admin" ? "Administrador" : "Usuário"}
                    </Badge>
                    {!item.has_password && <Badge variant="outline">Só Google</Badge>}
                  </CardDescription>
                </div>
                <CardAction className="flex gap-1">
                  <Button variant="ghost" size="icon-sm" aria-label={`Editar ${item.name}`} onClick={() => setDrawer({ open: true, user: item })}>
                    <PencilIcon />
                  </Button>
                  {item.id !== me?.id && (
                    <ConfirmDelete
                      trigger={
                        <Button variant="ghost" size="icon-sm" aria-label={`Apagar ${item.name}`}>
                          <Trash2Icon />
                        </Button>
                      }
                      title={`Apagar ${item.name}?`}
                      description="A pessoa perde o acesso na hora."
                      onConfirm={() => apagar(item)}
                    />
                  )}
                </CardAction>
              </CardHeader>
            </Card>
          </li>
        ))}
      </ul>

      <UserFormDrawer open={drawer.open} onOpenChange={(open) => setDrawer((state) => ({ ...state, open }))} user={drawer.user} />
    </>
  )
}
