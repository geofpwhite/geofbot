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
    {:ok, start_geofbot(state)}
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
      Port.close(state.port)
      {:noreply, Map.delete(state, :port)}
      {:noreply, start_geofbot(state)}
    else
      IO.puts("Geofbot output: #{data}")
      {:noreply, state}
    end
  end

  defp start_geofbot(state) do
    # Start the geofbot executable
    port =
      Port.open({:spawn_executable, "../geofbot"}, [
        :binary,
        args: ["-app=$appid", "-token=$bottoken", "-guild=$guildid"]
      ])

    IO.inspect(port)
    Port.monitor(port)
    # Process.monitor(port)
    Map.put(state, :port, port)
  end
end
