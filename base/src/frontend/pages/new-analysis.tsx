import { useState, type FormEvent } from "react"
import { FileSpreadsheetIcon, UploadIcon } from "lucide-react"
import { Link, useSearchParams } from "react-router"

import { RowsResult, SampleResult } from "@/components/analysis-result"
import { EmptyState } from "@/components/empty-state"
import { ErrorAlert } from "@/components/error-alert"
import { PageHeader } from "@/components/page-header"
import { ReportButton } from "@/components/report-button"
import { SubmitButton } from "@/components/submit-button"
import { TextField } from "@/components/text-field"
import { Button, buttonVariants } from "@/components/ui/button"
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ApiError } from "@/lib/api"
import { formatBytes, STATES } from "@/lib/format"
import { useCreateAnalysis, useFarms, useModels, useTanks } from "@/lib/queries"
import type { Analysis, Sample } from "@/lib/types"
import { cn } from "@/lib/utils"

const NONE = "none"
const DEFAULT_MODEL = "default"

const today = () => new Date().toISOString().slice(0, 10)

export default function NewAnalysisPage() {
  const [params] = useSearchParams()
  const farms = useFarms()

  // a fazenda: a da URL (?fazenda=3), a que a pessoa escolher, ou a única que existe
  const [chosenFarm, setChosenFarm] = useState(params.get("fazenda") ?? "")
  const farmList = farms.data ?? []
  const selected = farmList.find((item) => String(item.id) === chosenFarm) ?? (farmList.length === 1 ? farmList[0] : undefined)
  const id = selected?.id ?? 0

  const tanks = useTanks(id)
  const models = useModels()
  const create = useCreateAnalysis(id)

  const [mode, setMode] = useState<"sample" | "file">("sample")
  const [tankId, setTankId] = useState(NONE)
  const [model, setModel] = useState(DEFAULT_MODEL)
  const [file, setFile] = useState<File | null>(null)
  const [result, setResult] = useState<Analysis | null>(null)

  const [temperature, setTemperature] = useState("")
  const [ccs, setCcs] = useState("")
  const [cpp, setCpp] = useState("")
  const [date, setDate] = useState(today())
  const [state, setState] = useState("")
  const [city, setCity] = useState("")
  const [tankCode, setTankCode] = useState("")

  // a cidade e o estado da amostra começam com os da fazenda
  const cityValue = city || selected?.city || ""
  const stateValue = state || selected?.state || ""

  const fields = create.error instanceof ApiError ? create.error.fields : {}
  const fieldError = (name: string) => fields[`sample.${name}`] ?? fields[name]

  const tankItems = [{ value: NONE, label: "Sem tanque" }, ...(tanks.data ?? []).map((tank) => ({ value: String(tank.id), label: tank.type }))]
  const modelItems = [{ value: DEFAULT_MODEL, label: "Modelo padrão" }, ...(models.data?.models ?? []).map((name) => ({ value: name, label: name }))]

  const sampleReady = temperature && ccs && cpp && date && stateValue && cityValue.trim() && tankCode.trim()
  const ready = id > 0 && (mode === "sample" ? Boolean(sampleReady) : file !== null)

  async function submit(event: FormEvent) {
    event.preventDefault()

    const common = { tankId: tankId === NONE ? null : Number(tankId), model: model === DEFAULT_MODEL ? "" : model }
    const sample: Sample = {
      temperature: Number(temperature.replace(",", ".")),
      ccs: Number(ccs.replace(",", ".")),
      cpp: Number(cpp.replace(",", ".")),
      data: date,
      state: stateValue,
      cidade: cityValue.trim(),
      tanque: tankCode.trim(),
    }

    try {
      setResult(await create.mutateAsync(mode === "file" && file ? { ...common, file } : { ...common, sample }))
    } catch {
      // o erro aparece no formulário (create.error)
    }
  }

  function again() {
    setResult(null)
    setFile(null)
    create.reset()
  }

  if (result) {
    return (
      <>
        <PageHeader title="Resultado da análise" description={selected?.name} />
        {result.rows ? <RowsResult rows={result.rows} /> : <SampleResult result={result.result ?? ""} />}
        <div className="flex flex-col gap-3 sm:flex-row">
          <Button size="lg" className="sm:flex-1" onClick={again}>
            Fazer outra análise
          </Button>
          <ReportButton
            size="lg"
            className="sm:flex-1"
            data={() => ({
              analysis: result,
              farmName: selected?.name ?? `#${result.farm_id}`,
              tank: tanks.data?.find((item) => item.id === result.tank_id),
              contents: result.content ? [result.content] : [],
              rows: result.rows,
            })}
          />
          <Link to="/analises" className={cn(buttonVariants({ variant: "outline", size: "lg" }), "sm:flex-1")}>
            Ver minhas análises
          </Link>
        </div>
      </>
    )
  }

  return (
    <>
      <PageHeader title="Nova análise" description="O modelo diz se o leite está aprovado ou reprovado." back="/analises" />

      {farms.isError && <ErrorAlert error={farms.error} title="Não foi possível carregar as fazendas" />}

      {farms.isPending ? (
        <Skeleton className="h-64 w-full rounded-xl" />
      ) : farmList.length === 0 ? (
        <EmptyState
          title="Cadastre uma fazenda primeiro"
          description="Toda análise é de uma fazenda. Cadastre a sua e volte aqui."
          action={
            <Link to="/fazendas" className={cn(buttonVariants({ size: "lg" }))}>
              Ir para as fazendas
            </Link>
          }
        />
      ) : (
        <form onSubmit={submit} noValidate>
          <FieldGroup>
            {create.error && Object.keys(fields).length === 0 && <ErrorAlert error={create.error} title="Não foi possível analisar" />}

            <Field>
              <FieldLabel htmlFor="farm">Fazenda</FieldLabel>
              <Select
                value={selected ? String(selected.id) : ""}
                onValueChange={(value) => {
                  setChosenFarm(value ?? "")
                  // o tanque e a cidade são da fazenda: trocar de fazenda recomeça por eles
                  setTankId(NONE)
                  setCity("")
                  setState("")
                }}
                items={farmList.map((item) => ({ value: String(item.id), label: item.name }))}
              >
                <SelectTrigger id="farm" className="w-full">
                  <SelectValue placeholder="Escolha a fazenda" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {farmList.map((item) => (
                      <SelectItem key={item.id} value={String(item.id)}>
                        {item.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              <FieldDescription>De qual fazenda é o leite desta análise.</FieldDescription>
            </Field>

            <Tabs value={mode} onValueChange={(value) => setMode(value as "sample" | "file")}>
              <TabsList className="w-full">
                <TabsTrigger value="sample">Uma amostra</TabsTrigger>
                <TabsTrigger value="file">Um arquivo</TabsTrigger>
              </TabsList>

              <TabsContent value="sample" className="pt-4">
                <FieldGroup>
                  <TextField
                    name="temperature"
                    label="Temperatura (°C)"
                    type="number"
                    inputMode="decimal"
                    step="any"
                    value={temperature}
                    onChange={(event) => setTemperature(event.target.value)}
                    error={fieldError("temperature")}
                  />
                  <div className="grid grid-cols-2 gap-4">
                    <TextField name="ccs" label="CCS" type="number" inputMode="decimal" step="any" value={ccs} onChange={(event) => setCcs(event.target.value)} error={fieldError("ccs")} />
                    <TextField name="cpp" label="CPP" type="number" inputMode="decimal" step="any" value={cpp} onChange={(event) => setCpp(event.target.value)} error={fieldError("cpp")} />
                  </div>
                  <TextField name="data" label="Data da coleta" type="date" value={date} onChange={(event) => setDate(event.target.value)} error={fieldError("data")} />
                  <TextField
                    name="tanque"
                    label="Código do tanque"
                    description="Como aparece no laudo, ex.: TC01."
                    autoCapitalize="characters"
                    autoComplete="off"
                    value={tankCode}
                    onChange={(event) => setTankCode(event.target.value)}
                    error={fieldError("tanque")}
                  />
                  <div className="grid grid-cols-[1fr_6.5rem] gap-4">
                    <TextField name="cidade" label="Cidade" autoComplete="off" value={cityValue} onChange={(event) => setCity(event.target.value)} error={fieldError("cidade")} />
                    <Field data-invalid={fieldError("state") ? true : undefined}>
                      <FieldLabel htmlFor="state">UF</FieldLabel>
                      <Select value={stateValue} onValueChange={(value) => setState(value ?? "")} items={STATES.map((uf) => ({ value: uf, label: uf }))}>
                        <SelectTrigger id="state" className="w-full">
                          <SelectValue placeholder="UF" />
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
                      {fieldError("state") && <FieldError>{fieldError("state")}</FieldError>}
                    </Field>
                  </div>
                </FieldGroup>
              </TabsContent>

              <TabsContent value="file" className="pt-4">
                <Field data-invalid={fields.file ? true : undefined}>
                  <FieldLabel htmlFor="file">Arquivo de coletas</FieldLabel>
                  <label
                    htmlFor="file"
                    className="flex cursor-pointer flex-col items-center gap-2 rounded-xl border border-dashed p-6 text-center transition-colors hover:bg-accent/40 has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/50"
                  >
                    {file ? <FileSpreadsheetIcon className="size-8 text-primary" /> : <UploadIcon className="size-8 text-muted-foreground" />}
                    <span className="text-sm font-medium break-all">{file ? file.name : "Toque para escolher o arquivo"}</span>
                    <span className="text-xs text-muted-foreground">{file ? formatBytes(file.size) : "CSV (separado por ;) ou Parquet, até 20 MB"}</span>
                    <Input
                      id="file"
                      name="file"
                      type="file"
                      accept=".csv,.parquet"
                      className="sr-only"
                      onChange={(event) => setFile(event.target.files?.[0] ?? null)}
                    />
                  </label>
                  {fields.file ? <FieldError>{fields.file}</FieldError> : <FieldDescription>Cada linha do arquivo vira uma resposta do modelo.</FieldDescription>}
                </Field>
              </TabsContent>
            </Tabs>

            <Field>
              <FieldLabel htmlFor="tank">Tanque cadastrado</FieldLabel>
              <Select value={tankId} onValueChange={(value) => setTankId(value ?? NONE)} items={tankItems}>
                <SelectTrigger id="tank" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {tankItems.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              <FieldDescription>Opcional: liga a análise a um tanque desta fazenda.</FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor="model">Modelo</FieldLabel>
              <Select value={model} onValueChange={(value) => setModel(value ?? DEFAULT_MODEL)} items={modelItems}>
                <SelectTrigger id="model" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {modelItems.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              {models.isError && <FieldDescription>A lista de modelos não carregou. Vamos usar o modelo padrão.</FieldDescription>}
            </Field>

            <SubmitButton size="lg" pending={create.isPending} pendingLabel="Analisando" disabled={!ready}>
              Analisar leite
            </SubmitButton>
          </FieldGroup>
        </form>
      )}
    </>
  )
}
