import type { ReactNode } from "react"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"

interface Props {
  /** O botão que abre a confirmação. */
  trigger: ReactNode
  title: string
  description: string
  onConfirm: () => void
  /** O texto do botão que confirma (padrão "Apagar"). */
  confirmLabel?: string
  /** "destructive" (vermelho, o padrão) para apagar; "default" para ações como sair. */
  variant?: "destructive" | "default"
}

/** Pede confirmação antes de uma ação difícil de desfazer (apagar, sair). */
export function ConfirmDelete({ trigger, title, description, onConfirm, confirmLabel = "Apagar", variant = "destructive" }: Props) {
  return (
    <AlertDialog>
      <AlertDialogTrigger render={trigger as React.ReactElement} />
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancelar</AlertDialogCancel>
          <AlertDialogAction variant={variant} onClick={onConfirm}>
            {confirmLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
