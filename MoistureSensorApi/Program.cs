using System.Net;
using System.Text;
using System.Text.Json;
using Amazon;
using Amazon.IotData;
using Amazon.IotData.Model;
using Amazon.Runtime;
using Amazon.Runtime.CredentialManagement;
using HttpContext = Microsoft.AspNetCore.Http.HttpContext;

namespace MoistureSensorApi;

public static class Program
{
    // create an aws iot client
    private static readonly CredentialProfileStoreChain Chain = new();
    private static readonly AWSCredentials Credentials = null!;
    private static bool _ = Chain.TryGetAWSCredentials("aws-osuapp", out Credentials);
    private const string ServiceUrl = "https://a3qga117xn0bd5-ats.iot.us-west-2.amazonaws.com";

    // check if in development
    private static readonly bool InDevelopment = Environment.GetEnvironmentVariable("ASPNETCORE_ENVIRONMENT") == "Development";
    private static readonly AmazonIotDataClient IotClient = InDevelopment ? new AmazonIotDataClient(ServiceUrl, Credentials) : new AmazonIotDataClient(ServiceUrl);

    public static void Main(string[] args)
    {
        var builder = WebApplication.CreateBuilder(args);

        // Add services to the container.
        builder.Services.AddAuthorization();

        // Learn more about configuring Swagger/OpenAPI at https://aka.ms/aspnetcore/swashbuckle
        builder.Services.AddEndpointsApiExplorer();
        builder.Services.AddSwaggerGen();
        // NOTE: swagger docs won't work with AWS Lambda hosting. You'll have to upgrade to full ASP.NET Core API.

        // Add AWS Lambda hosting
        builder.Services.AddAWSLambdaHosting(LambdaEventSource.HttpApi);

        var app = builder.Build();

        // Configure the HTTP request pipeline.
        // if (app.Environment.IsDevelopment())
        // {
            app.UseSwagger();
            app.UseSwaggerUI();
        // }

        app.UseHttpsRedirection();

        app.UseAuthorization();
        
        // https://learn.microsoft.com/en-us/aspnet/core/fundamentals/logging/?view=aspnetcore-7.0
        var logger = LoggerFactory.Create(config =>
        {
            config.AddConsole();
        }).CreateLogger("Program");
        
        logger.LogInformation("In development: {IsDevelopment}", InDevelopment);

        app.MapGet("/report-data/{deviceId}", async (HttpContext httpContext, string deviceId, int temperature, int pressure, int moisture) =>
        {
                var sensorData = new SensorData
                {
                    DeviceId = deviceId,
                    Temperature = temperature,
                    Pressure = pressure,
                    Moisture = moisture
                };
                
                try
                {
                    // make memory stream in format { "state": { "reported": { "temperature": 0, "pressure": 0, "moisture": 0 } } }
                    var shadow = new MemoryStream(Encoding.UTF8.GetBytes(JsonSerializer.Serialize(new
                    {
                        state = new
                        {
                            reported = new
                            {
                                temperature = sensorData.Temperature,
                                pressure = sensorData.Pressure,
                                moisture = sensorData.Moisture
                            }
                        }
                    })));

                    var response = await UpdateShadow(deviceId, shadow);
                    
                    // log the response
                    logger.LogInformation("Update shadow response: {Response}", response);
                }
                catch (Exception e)
                {
                    // log the error
                    logger.LogError("Failed to update shadow: {Error}", e);
                    // return json in format: { "error": "Failed to update shadow" }
                    return Results.BadRequest(new
                    {
                        error = e.Message,
                        type = e.GetType().Name,
                        trace = e.StackTrace
                    });
                }

                return Results.Ok(sensorData);
            });

        app.Run();
    }

    private static async Task<string> UpdateShadow(string deviceId, MemoryStream shadow)
    {
        // create a request to update the shadow
        var updateShadowRequest = new UpdateThingShadowRequest
        {
            ThingName = deviceId,
            Payload = shadow
        };
        
        // send the request
        var updateShadowResponse = await IotClient.UpdateThingShadowAsync(updateShadowRequest);
        
        // check the response
        // NOTE: doesn't seem to throw error even if deviceId doesn't exist in AWS IoT
        if (updateShadowResponse.HttpStatusCode != HttpStatusCode.OK)
        {
            throw new Exception("Failed to update shadow");
        }
        
        // read out memory stream
        var responseString = await new StreamReader(updateShadowResponse.Payload).ReadToEndAsync();

        return responseString;
    }
}

