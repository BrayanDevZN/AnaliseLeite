import type { ComponentProps } from "react"

import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"

interface Props extends Omit<ComponentProps<typeof Button>, "type"> {
  pending?: boolean
  pendingLabel?: string
}

/** Botão de enviar formulário: mostra um spinner e trava enquanto a chamada está em andamento. */
export function SubmitButton({ pending, pendingLabel, children, disabled, ...props }: Props) {
  return (
    <Button type="submit" disabled={pending || disabled} {...props}>
      {pending && <Spinner data-icon="inline-start" />}
      {pending ? (pendingLabel ?? children) : children}
    </Button>
  )
}
