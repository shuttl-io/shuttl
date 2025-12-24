using Shuttl.Models;
using Shuttl.Models.Tools;
using Shuttl.Models.Secrets;

namespace Shuttl.Test;

/// <summary>
/// C# test application for integration testing
///
/// This app creates a simple agent with a toolkit and tool,
/// then starts the StdInServer to accept IPC commands.
/// </summary>
public class Program
{
    public static void Main(string[] args)
    {
        // Create server
        var server = new StdInServer();
        
        // Create app
        var app = new App("DotNetTestApp", server);
        
        // Create model
        var model = new Model(new ModelProps
        {
            Identifier = "test-model",
            Key = Secret.fromEnv("OPENAI_API_KEY")
        });
        
        // Create toolkit with tools
        var utilToolkit = new Toolkit(new ToolkitProps
        {
            Name = "UtilityToolkit",
            Description = "A toolkit with utility functions",
            Tools = new ITool[] { new EchoTool(), new MathTool() }
        });
        
        // Create agent
        var agent = new Agent(new AgentProps
        {
            Name = "TestAgent",
            SystemPrompt = "You are a helpful test agent for integration testing.",
            Model = model,
            Toolkits = new Toolkit[] { utilToolkit }
        });
        
        // Add agent to app
        app.AddAgent(agent);
        
        // Start serving
        app.Serve();
    }
}

/// <summary>
/// A simple test tool that echoes its input
/// </summary>
public class EchoTool : ITool
{
    public string Name { get; set; } = "echo";
    public string Description { get; set; } = "Echoes the input message back";

    public object Execute(IDictionary<string, object> args)
    {
        var message = args.TryGetValue("message", out var msg) ? msg?.ToString() : "no message";
        return new Dictionary<string, object>
        {
            ["echoed"] = message ?? "no message",
            ["timestamp"] = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds()
        };
    }

    public IDictionary<string, ToolArg> ProduceArgs()
    {
        return new Dictionary<string, ToolArg>
        {
            ["message"] = new ToolArg
            {
                Name = "message",
                ArgType = "string",
                Description = "The message to echo",
                Required = true,
                DefaultValue = null
            }
        };
    }
}

/// <summary>
/// A tool that performs simple math
/// </summary>
public class MathTool : ITool
{
    public string Name { get; set; } = "add";
    public string Description { get; set; } = "Adds two numbers together";

    public object Execute(IDictionary<string, object> args)
    {
        var a = args.TryGetValue("a", out var aVal) ? Convert.ToDouble(aVal) : 0;
        var b = args.TryGetValue("b", out var bVal) ? Convert.ToDouble(bVal) : 0;
        return new Dictionary<string, object>
        {
            ["result"] = a + b,
            ["operation"] = "add"
        };
    }

    public IDictionary<string, ToolArg> ProduceArgs()
    {
        return new Dictionary<string, ToolArg>
        {
            ["a"] = new ToolArg
            {
                Name = "a",
                ArgType = "number",
                Description = "First number",
                Required = true,
                DefaultValue = null
            },
            ["b"] = new ToolArg
            {
                Name = "b",
                ArgType = "number",
                Description = "Second number",
                Required = true,
                DefaultValue = null
            }
        };
    }
}






















