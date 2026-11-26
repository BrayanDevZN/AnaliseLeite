import { useState, type FormEvent } from "react"

import { ErrorAlert } from "@/components/error-alert"
import { FormDrawer } from "@/components/form-drawer"
import { SubmitButton } from "@/components/submit-button"
import { TextField } from "@/components/text-field"
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { ApiError } from "@/lib/api"
import { STATES } from "@/lib/format"
import { notify } from "@/lib/notify"
import { useCreateFarm, useUpdateFarm } from "@/lib/queries"
import type { Farm } from "@/lib/types"

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Com uma fazenda, edita ela; sem, cria uma nova. */
  farm?: Farm
  /** Chamado com a fazenda criada ou editada, depois de salvar. */
  onSaved?: (farm: Farm) => void
}

const items = STATES.map((state) => ({ value: state, label: state }))

export function FarmFormDrawer({ open, onOpenChange, farm, onSaved }: Props) {
  return (
    <FormDrawer open={open} onOpenChange={onOpenChange} title={farm ? "Editar fazenda" : "Nova fazenda"} description="Onde o leite é produzido.">
      {/* o formulário só existe com a folha aberta: nasce novo (campos e erros zerados) a cada abertura */}
      <FarmForm farm={farm} onDone={() => onOpenChange(false)} onSaved={onSaved} />
    </FormDrawer>
  )
}

function FarmForm({ farm, onDone, onSaved }: { farm?: Farm; onDone: () => void; onSaved?: (farm: Farm) => void }) {
  const create = useCreateFarm()
  const update = useUpdateFarm(farm?.id ?? 0)
  const mutation = farm ? update : create

  const [name, setName] = useState(farm?.name ?? "")
  const [city, setCity] = useState(farm?.city ?? "")
  const [state, setState] = useState(farm?.state ?? "")

  const fields = mutation.error instanceof ApiError ? mutation.error.fields : {}

  async function submit(event: FormEvent) {
    event.preventDefault()

    try {
      const saved = await mutation.mutateAsync({ name: name.trim(), city: city.trim(), state })

      notify.success(farm ? "Fazenda atualizada" : "Fazenda criada")
      onDone()
      onSaved?.(saved)
    } catch {
      // o erro já aparece no formulário (mutation.error)
    }
  }

  return (
    <form onSubmit={submit} noValidate>
      <FieldGroup>
        {mutation.error && Object.keys(fields).length === 0 && <ErrorAlert error={mutation.error} />}

        <TextField name="name" label="Nome da fazenda" autoComplete="off" value={name} onChange={(event) => setName(event.target.value)} error={fields.name} required />
        <TextField name="city" label="Cidade" autoComplete="off" value={city} onChange={(event) => setCity(event.target.value)} error={fields.city} required />

        <Field data-invalid={fields.state ? true : undefined}>
          <FieldLabel htmlFor="state">Estado</FieldLabel>
          <Select value={state} onValueChange={(value) => setState(value ?? "")} items={items}>
            <SelectTrigger id="state" className="w-full" aria-invalid={fields.state ? true : undefined}>
              <SelectValue placeholder="Escolha o estado" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {STATES.map((uf) => (
                  <SelectItem key={uf} value={uf}>
                    {uf}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          {fields.state && <FieldError>{fields.state}</FieldError>}
        </Field>

        <SubmitButton size="lg" pending={mutation.isPending} pendingLabel="Salvando" disabled={!state}>
          Salvar
        </SubmitButton>
      </FieldGroup>
    </form>
  )
}
