import type { ReactNode } from "react"

import { GoogleIcon } from "@/components/google-icon"
import { buttonVariants } from "@/components/ui/button"
import { GOOGLE_LOGIN_URL } from "@/lib/api"
import { cn } from "@/lib/utils"

/**
 * Botão "com o Google". É um link, e não um fetch: o navegador vai para o backend, que redireciona para o Google
 * e, na volta, entra na conta (criando ela se for a primeira vez) e leva a pessoa para dentro do sistema.
 */
export function GoogleButton({ children }: { children: ReactNode }) {
  return (
    <a href={GOOGLE_LOGIN_URL} className={cn(buttonVariants({ variant: "outline", size: "lg" }))}>
      <GoogleIcon className="size-4" />
      {children}
    </a>
  )
}
