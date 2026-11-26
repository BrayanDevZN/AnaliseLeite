import type { ComponentProps } from "react"

import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

interface Props extends Omit<ComponentProps<typeof Input>, "id"> {
  /** Também é o id do campo e o nome que o backend usa nos erros (ex.: "name"). */
  name: string
  label: string
  description?: string
  /** Mensagem de erro deste campo (vinda do backend ou da tela). */
  error?: string
}

/** Campo de formulário completo: rótulo, campo, dica e erro, no padrão Field do shadcn. */
export function TextField({ name, label, description, error, ...props }: Props) {
  return (
    <Field data-invalid={error ? true : undefined}>
      <FieldLabel htmlFor={name}>{label}</FieldLabel>
      <Input id={name} name={name} aria-invalid={error ? true : undefined} {...props} />
      {description && !error && <FieldDescription>{description}</FieldDescription>}
      {error && <FieldError>{error}</FieldError>}
    </Field>
  )
}
