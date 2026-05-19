import { zodResolver } from "@hookform/resolvers/zod";
import { Lock, LogIn } from "lucide-react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { z } from "zod";

import { login } from "../services/api";
import { saveTokens } from "../services/session";

const schema = z.object({
  username: z.string().min(3, "Informe o usuário."),
  password: z.string().min(8, "A senha precisa ter ao menos 8 caracteres.")
});

type LoginForm = z.infer<typeof schema>;

export function LoginPage() {
  const navigate = useNavigate();
  const form = useForm<LoginForm>({
    resolver: zodResolver(schema),
    defaultValues: { username: "admin", password: "" }
  });

  async function onSubmit(values: LoginForm) {
    try {
      const tokens = await login(values.username, values.password);
      saveTokens(tokens.access_token, tokens.refresh_token);
      toast.success("Login realizado.");
      navigate("/");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Falha no login.");
    }
  }

  return (
    <main className="mx-auto flex min-h-[calc(100vh-64px)] max-w-md items-center px-4">
      <form className="w-full space-y-5" onSubmit={form.handleSubmit(onSubmit)}>
        <div>
          <div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <Lock className="h-5 w-5" />
          </div>
          <h1 className="text-2xl font-semibold">Entrar no painel</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Use o administrador criado na primeira inicialização.
          </p>
        </div>
        <label className="block space-y-2">
          <span className="text-sm font-medium">Usuário</span>
          <input className="field" {...form.register("username")} autoComplete="username" />
          <span className="error">{form.formState.errors.username?.message}</span>
        </label>
        <label className="block space-y-2">
          <span className="text-sm font-medium">Senha</span>
          <input
            className="field"
            {...form.register("password")}
            type="password"
            autoComplete="current-password"
          />
          <span className="error">{form.formState.errors.password?.message}</span>
        </label>
        <button className="button-primary w-full" type="submit" disabled={form.formState.isSubmitting}>
          <LogIn className="h-4 w-4" />
          {form.formState.isSubmitting ? "Entrando..." : "Entrar"}
        </button>
      </form>
    </main>
  );
}
