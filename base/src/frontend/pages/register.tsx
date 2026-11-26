import { useState, type FormEvent } from "react"
import { Link, useNavigate } from "react-router"

import { AuthLayout } from "@/components/auth-layout"
import { ErrorAlert } from "@/components/error-alert"
import { GoogleButton } from "@/components/google-button"
import { SubmitButton } from "@/components/submit-button"
import { TextField } from "@/components/text-field"
import { Card, CardContent } from "@/components/ui/card"
import { FieldGroup, FieldSeparator } from "@/components/ui/field"
import { ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"

export default function RegisterPage() {
  const { register, login } = useAuth()
  const navigate = useNavigate()

  const [name, setName] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<ApiError | Error | null>(null)

  const fields = error instanceof ApiError ? error.fields : {}

  async function submit(event: FormEvent) {
    event.preventDefault()
    setPending(true)
    setError(null)

    try {
      await register(name.trim(), email.trim(), password)
      await login(email.trim(), password)
      navigate("/fazendas", { replace: true })
    } catch (caught) {
      setError(caught instanceof Error ? caught : new Error(String(caught)))
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthLayout title="Crie sua conta" description="Leva menos de um minuto.">
      <Card className="w-full">
        <CardContent>
          <form onSubmit={submit} noValidate>
            <FieldGroup>
              {error && Object.keys(fields).length === 0 && <ErrorAlert error={error} title="Não foi possível cadastrar" />}

              <TextField name="name" label="Nome" autoComplete="name" value={name} onChange={(event) => setName(event.target.value)} error={fields.name} required />
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
                required
              />
              <TextField
                name="password"
                label="Senha"
                type="password"
                autoComplete="new-password"
                description="No mínimo 8 caracteres."
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                error={fields.password}
                required
              />

              <SubmitButton size="lg" pending={pending} pendingLabel="Criando conta">
                Criar conta
              </SubmitButton>

              <FieldSeparator>ou</FieldSeparator>

              {/* na primeira vez, entrar com o Google já cria a conta: não precisa preencher nada */}
              <GoogleButton>Cadastrar com o Google</GoogleButton>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>

      <p className="text-sm text-muted-foreground">
        Já tem conta?{" "}
        <Link to="/entrar" className="font-medium text-primary underline-offset-4 hover:underline">
          Entrar
        </Link>
      </p>
    </AuthLayout>
  )
}
