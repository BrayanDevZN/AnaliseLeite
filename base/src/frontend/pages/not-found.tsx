import { Link } from "react-router"

import { AuthLayout } from "@/components/auth-layout"
import { buttonVariants } from "@/components/ui/button"
import { cn } from "@/lib/utils"

export default function NotFoundPage() {
  return (
    <AuthLayout title="Página não encontrada" description="Essa vaca se perdeu do rebanho. O endereço não existe.">
      <Link to="/fazendas" className={cn(buttonVariants({ size: "lg" }))}>
        Voltar para as fazendas
      </Link>
    </AuthLayout>
  )
}
