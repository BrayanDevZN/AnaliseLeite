import { useState, type FormEvent } from "react"
import { Link, useSearchParams } from "react-router"

import { AuthLayout } from "@/components/auth-layout"
import { ErrorAlert } from "@/components/error-alert"
import { GoogleButton } from "@/components/google-button"
import { SubmitButton } from "@/components/submit-button"
import { TextField } from "@/components/text-field"
import { Card, CardContent } from "@/components/ui/card"
import { FieldGroup, FieldSeparator } from "@/components/ui/field"
import { ApiError } from "@/lib/api"
import { useAuth } from "@/lib/auth"
import { GOOGLE_ERRORS } from "@/lib/format"

export default function LoginPage() {
  const { login } = useAuth()
  const [params] = useSearchParams()

  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<unknown>(null)

  // o backend volta para cá com ?error=<motivo> quando o login com o Google dá errado
  const googleError = params.get("error")
  const googleMessage = googleError ? (GOOGLE_ERRORS[googleError] ?? "Não foi possível entrar com o Google.") : null

  async function submit(event: FormEvent) {
    event.preventDefault()
    setPending(true)
    setError(null)

    try {
      // depois do login, quem navega é o guarda PublicOnly (para o `from`, se houver, ou as fazendas)
      await login(email.trim(), password)
    } catch (caught) {
      // nas telas de entrar, 401 quer dizer "email ou senha errados"
      setError(caught instanceof ApiError && caught.status === 401 ? new Error("Email ou senha incorretos.") : caught)
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthLayout title="Entre na sua conta" description="Acompanhe suas fazendas, tanques e análises de leite.">
      <Card className="w-full">
        <CardContent>
          <form onSubmit={submit} noValidate>
            <FieldGroup>
              {googleMessage && <ErrorAlert error={new Error(googleMessage)} title="Não foi possível entrar" />}
              {error != null && <ErrorAlert error={error} title="Não foi possível entrar" />}

              <TextField
                name="email"
                label="Email"
                type="email"
                inputMode="email"
                autoComplete="email"
                autoCapitalize="none"
                placeholder="voce@exemplo.com"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                required
              />
              <TextField
                name="password"
                label="Senha"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                required
              />

              <SubmitButton size="lg" pending={pending} pendingLabel="Entrando">
                Entrar
              </SubmitButton>

              <FieldSeparator>ou</FieldSeparator>

              <GoogleButton>Entrar com o Google</GoogleButton>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>

      <p className="text-sm text-muted-foreground">
        Ainda não tem conta?{" "}
        <Link to="/cadastro" className="font-medium text-primary underline-offset-4 hover:underline">
          Cadastre-se
        </Link>
      </p>
    </AuthLayout>
  )
}
