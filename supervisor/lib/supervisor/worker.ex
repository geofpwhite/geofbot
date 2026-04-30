# defmodule Supervisor.Worker do
#   use GenServer

#   def start_link(_args) do
#     GenServer.start_link(__MODULE__, :ok, name: __MODULE__)
#   end

#   def init(:ok) do
#     {:ok, spawn_process()}
#   end

#   defp spawn_process do
#     Task.start_link(fn ->
#       System.cmd("../geofbot.exe", [], into: IO.stream(:stdio, :line))
#     end)
#   end

#   def handle_info({:EXIT, _pid, _reason}, state) do
#     {:noreply, spawn_process()}
#   end
# end
defmodule Supervisor.Worker do
  use GenServer

  def start_link(_args) do
    GenServer.start_link(__MODULE__, %{}, name: __MODULE__)
  end

  @impl true
  def init(state) do
    # Start the geofbot process
    p = start_geofbot(state)
    IO.inspect(p)
    {:ok, p}
  end

  @impl true
  def handle_info({:DOWN, _ref, :port, _pid, _reason}, state) do
    # Restart the geofbot process if it exits
    {:noreply, start_geofbot(state)}
  end

  def handle_info({_port, {:data, data}}, state) do
    # Handle output from the geofbot process
    if String.contains?(data, "Broken Pipe") do
      IO.puts("Broken Pipe detected. Stopping geofbot...")

      os_pid =
        case :erlang.port_info(state.port, :os_pid) do
          {:os_pid, pid} -> pid
          _ -> nil
        end

      if os_pid do
        # Send SIGINT
        :os.cmd("kill -SIGINT #{os_pid}")
        IO.puts("Sent SIGINT to PID #{os_pid}")
      end

      Port.close(state.port)
      {:noreply, start_geofbot( Map.delete(state, :port))}
    else
      IO.puts("Geofbot output: #{data}")
      {:noreply, state}
    end
  end

  defp start_geofbot(state) do
    # Start the geofbot executable
    env = System.get_env()
    appid = env["appid"] || "default_appid"
    bottoken = env["bottoken"] || "default_bottoken"
    guildid = env["guildid"] || "default_guildid"
    port =
      Port.open({:spawn_executable, "./geofbot"}, [
        :binary,
        args: ["-app="<>appid, "-token="<>bottoken, "-guild="<>guildid]
      ])

    IO.inspect(port)
    Port.monitor(port)
    # Process.monitor(port)

    Map.put(state, :port, port)
  end
end
