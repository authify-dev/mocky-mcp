import asyncio
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from agents import Agent, Runner
from agents.mcp import MCPServerStdio
from agents.run_context import RunContextWrapper

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 🚀 Inicializamos el MCP server y el agente globalmente
mcp_server: MCPServerStdio | None = None
agent: Agent | None = None

class MessageRequest(BaseModel):
    text: str

@app.on_event("startup")
async def startup_event():
    global mcp_server, agent

    # Arrancamos el servidor MCP (Go)
    mcp_server = MCPServerStdio(
        params={"command": "./server-mcp"},
    )
    await mcp_server.connect()

    # Ver tools disponibles
    run_ctx = RunContextWrapper(context=None)
    probe_agent = Agent(name="probe", instructions="probe")
    tools = await mcp_server.list_tools(run_ctx, probe_agent)
    print("🔎 MCP tools:", [t.name for t in tools])

    # Configurar agente
    agent = Agent(
        name="Asistente",
        instructions=(
            "Reglas:\n"
            "- Si el usuario pide un saludo o da un nombre, "
            "usa la tool MCP 'hello_world' con el argumento 'name'.\n"
            "- Si el usuario pide ver prototipos, endpoints, "
            "o listar nombres/métodos/urlPaths, "
            "usa la tool MCP 'list_prototypes'.\n"
            "- Si el usuario pide ver detalles de un prototipo, "
            "usa la tool MCP 'get_prototype_detail' con el argumento 'id' o 'name'.\n"
            "- No inventes resultados; responde exclusivamente con "
            "la salida de la tool MCP."
        ),
        mcp_servers=[mcp_server],
        model="gpt-4.1-mini",
    )

@app.on_event("shutdown")
async def shutdown_event():
    if mcp_server:
        await mcp_server.cleanup()

# 📩 Endpoint para enviar mensajes
@app.post("/message/send")
async def send_message(req: MessageRequest):
    if agent is None:
        return {"error": "Agent not initialized"}

    result = await Runner.run(agent, req.text)
    output = getattr(result, "final_output", None) or getattr(result, "output_text", "")
    return {"response": output}
