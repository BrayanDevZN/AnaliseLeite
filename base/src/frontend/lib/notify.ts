import { toast } from "@/components/ui/toast"

/** Avisos rápidos na tela (toast). */
export const notify = {
  success: (title: string, description?: string) => toast.add({ title, description, type: "success", timeout: 4000 }),
  error: (title: string, description?: string) => toast.add({ title, description, type: "error", timeout: 6000 }),
}
