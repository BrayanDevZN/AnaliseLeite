import { useState, type FormEvent } from "react"

import { ErrorAlert } from "@/components/error-alert"
import { FormDrawer } from "@/components/form-drawer"
import { SubmitButton } from "@/components/submit-button"
import { TextField } from "@/components/text-field"
import { FieldGroup } from "@/components/ui/field"
import { ApiError } from "@/lib/api"
import { notify } from "@/lib/notify"
import { useCreateTank, useUpdateTank } from "@/lib/queries"
import type { Tank } from "@/lib/types"

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  farmId: number
  /** Com um tanque, edita ele; sem, cria um novo. */
  tank?: Tank
}

export function TankFormDrawer({ open, onOpenChange, farmId, tank }: Props) {
  return (
    <FormDrawer open={open} onOpenChange={onOpenChange} title={tank ? "Editar tanque" : "Novo tanque"} description="Onde o leite fica guardado até a coleta.">
      {/* o formulário só existe com a folha aberta: nasce novo a cada abertura */}
      <TankForm farmId={farmId} tank={tank} onDone={() => onOpenChange(false)} />
    </FormDrawer>
  )
}

function TankForm({ farmId, tank, onDone }: { farmId: number; tank?: Tank; onDone: () => void }) {
  const create = useCreateTank(farmId)
  const update = useUpdateTank(farmId, tank?.id ?? 0)
  const mutation = tank ? update : create

  const [type, setType] = useState(tank?.type ?? "")
  const [capacity, setCapacity] = useState(tank ? String(tank.capacity_liters) : "")

  const fields = mutation.error instanceof ApiError ? mutation.error.fields : {}

  async function submit(event: FormEvent) {
    event.preventDefault()

    try {
      await mutation.mutateAsync({ type: type.trim(), capacity_liters: Number(capacity.replace(",", ".")) })

      notify.success(tank ? "Tanque atualizado" : "Tanque criado")
      onDone()
    } catch {
      // o erro já aparece no formulário (mutation.error)
    }
  }

  return (
    <form onSubmit={submit} noValidate>
      <FieldGroup>
        {mutation.error && Object.keys(fields).length === 0 && <ErrorAlert error={mutation.error} />}

        <TextField
          name="type"
          label="Tipo do tanque"
          description="Ex.: expansão, comunitário."
          autoComplete="off"
          value={type}
          onChange={(event) => setType(event.target.value)}
          error={fields.type}
          required
        />
        <TextField
          name="capacity_liters"
          label="Capacidade (litros)"
          type="number"
          inputMode="decimal"
          min="1"
          step="any"
          value={capacity}
          onChange={(event) => setCapacity(event.target.value)}
          error={fields.capacity_liters}
          required
        />

        <SubmitButton size="lg" pending={mutation.isPending} pendingLabel="Salvando" disabled={!type.trim() || !capacity}>
          Salvar
        </SubmitButton>
      </FieldGroup>
    </form>
  )
}
