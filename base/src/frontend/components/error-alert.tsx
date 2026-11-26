import { TriangleAlertIcon } from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"

/** Mostra o erro de uma chamada (ou de uma tela) de forma visível, sem tirar o usuário da página. */
export function ErrorAlert({ error, title = "Algo deu errado" }: { error: unknown; title?: string }) {
  if (!error) return null

  const message = error instanceof Error ? error.message : String(error)

  return (
    <Alert variant="destructive" role="alert">
      <TriangleAlertIcon />
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  )
}
