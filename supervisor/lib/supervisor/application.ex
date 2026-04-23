defmodule Supervisor.Application do
  use Application

  def start(_type, _args) do
    children = [
      {Supervisor.Worker, []}
    ]

    opts = [strategy: :one_for_one, name: Supervisor.Supervisor]
    Supervisor.start_link(children, opts)
  end

  def main(["-l"]) do
    env = System.get_env()
    appid = env["appid"] || "default_appid"
    bottoken = env["bottoken"] || "default_bottoken"
    guildid = env["guildid"] || "default_guildid"
    IO.inspect(appid)
    IO.inspect(bottoken)
    IO.inspect(guildid)
  end

  def main(_args \\ []) do
    # Start the application
    {:ok, _pid} = Supervisor.Application.start(:normal, [])

    # Keep the main process alive
    Process.sleep(:infinity)
  end
end
