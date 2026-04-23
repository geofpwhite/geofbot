defmodule Supervisor.Application do
  use Application

  def start(_type, _args) do
    children = [
      {Supervisor.Worker, []}
    ]

    opts = [strategy: :one_for_one, name: Supervisor.Supervisor]
    Supervisor.start_link(children, opts)
  end
end