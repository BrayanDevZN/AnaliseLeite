import type { ReactNode } from "react"

import { Drawer, DrawerContent, DrawerDescription, DrawerHeader, DrawerTitle } from "@/components/ui/drawer"

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  children: ReactNode
}

/** Folha que sobe de baixo (no celular dá para arrastar para fechar) com um formulário dentro. */
export function FormDrawer({ open, onOpenChange, title, description, children }: Props) {
  return (
    <Drawer open={open} onOpenChange={onOpenChange}>
      <DrawerContent>
        <DrawerHeader>
          <DrawerTitle>{title}</DrawerTitle>
          {description && <DrawerDescription>{description}</DrawerDescription>}
        </DrawerHeader>
        <div className="overflow-y-auto p-4 pb-[max(1rem,env(safe-area-inset-bottom))]">{children}</div>
      </DrawerContent>
    </Drawer>
  )
}
