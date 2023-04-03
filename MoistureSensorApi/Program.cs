using System.Net;
using System.Text;
using System.Text.Json;
using Amazon;
using Amazon.IotData;
using Amazon.IotData.Model;
using Amazon.Runtime;
using Amazon.Runtime.CredentialManagement;
using Amazon.TimestreamQuery;
using Amazon.TimestreamQuery.Model;
using Newtonsoft.Json;
using HttpContext = Microsoft.AspNetCore.Http.HttpContext;
using JsonSerializer = System.Text.Json.JsonSerializer;

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


    // https://learn.microsoft.com/en-us/aspnet/core/fundamentals/logging/?view=aspnetcore-7.0
    private static readonly ILogger Logger = LoggerFactory.Create(config =>
    {
        config.AddConsole();
    }).CreateLogger("MoistureSensorApi");

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
        
        Logger.LogInformation("In development: {IsDevelopment}", InDevelopment);
        
        app.MapGet("/fetch-data/{deviceId}", async (HttpContext httpContext, string deviceId) =>
            {
                try
                {
                    // print out device id
                    Logger.LogInformation("Device ID: {DeviceId}", deviceId);
                    
                    dynamic responseJson = await HttpRequest(deviceId) ?? throw new InvalidOperationException();
                    
                    // parse responseJson.data.getSensor
                    var getSensor = responseJson.data.getSensor;
                    
                    // check if device exists
                    if (getSensor == null)
                    {
                        throw new Exception("Device does not exist");
                    }
                    
                    // get thing name
                    var thingName = getSensor.thingName.Value as string;
                    Logger.LogInformation("Thing name: {ThingName}", thingName);
                    if (thingName == null)
                    {
                        throw new Exception("Thing name is null");
                    }
                    
                    // get sensor data from timestream
                    var sensorData = await GetSensorData(thingName);

                    return Results.Ok(sensorData);
                }
                catch (Exception e)
                {
                    // log the error
                    Logger.LogError("Error: {Error}", e);
                    // return json in format: { "error": "Failed to update shadow" }
                    return Results.BadRequest(new
                    {
                        error = e.Message,
                        type = e.GetType().Name,
                        trace = e.StackTrace
                    });
                }
            });

        app.MapGet("/report-data/{deviceId}", async (HttpContext httpContext, string deviceId, int temperature, int pressure, int moisture) =>
            {
                var sensorData = new SensorDataInput
                {
                    Temperature = temperature,
                    Pressure = pressure,
                    Moisture = moisture
                };
                
                try
                {
                    // print out device id
                    Logger.LogInformation("Device ID: {DeviceId}", deviceId);
                    
                    dynamic responseJson = await HttpRequest(deviceId) ?? throw new InvalidOperationException();
                    
                    // parse responseJson.data.getSensor
                    var getSensor = responseJson.data.getSensor;
                    
                    // check if device exists
                    if (getSensor == null)
                    {
                        throw new Exception("Device does not exist");
                    }
                    
                    // get thing name
                    var thingName = getSensor.thingName.Value as string;
                    Logger.LogInformation("Thing name: {ThingName}", thingName);
                    if (thingName == null)
                    {
                        throw new Exception("Thing name is null");
                    }

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

                    var response = await UpdateShadow(thingName, shadow);
                    
                    // log the response
                    Logger.LogInformation("Update shadow response: {Response}", response);
                }
                catch (Exception e)
                {
                    // log the error
                    Logger.LogError("Error: {Error}", e);
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

    private static async Task<object?> HttpRequest(string deviceId)
    {
        // make an http request to a graphql endpoint with x-api-key header
        var request = new HttpRequestMessage(HttpMethod.Post, "https://7h6nr2h6n5amtaadd5db7gbu2i.appsync-api.us-west-2.amazonaws.com/graphql");
        
        // add the x-api-key header
        request.Headers.Add("x-api-key", "da2-gnn7q3s2izhrnis7hypn3zt7ue");
        
        // add the body
        request.Content = new StringContent(JsonSerializer.Serialize(new
        {
            query = $"query MyQuery {{ getSensor(id: \"{deviceId}\") {{ thingName }} }}"
        }), Encoding.UTF8, "application/json");
        
        // print out the request data
        Logger.LogInformation("Get device request: {Request}", await request.Content.ReadAsStringAsync());
        
        // send the request
        var response = await new HttpClient().SendAsync(request);
        
        // check the response
        if (response.StatusCode != HttpStatusCode.OK)
        {
            throw new Exception("Status code not OK: " + response.StatusCode);
        }
        
        // read the response
        var responseString = await response.Content.ReadAsStringAsync();
        
        // log the response
        Logger.LogInformation("Get device response: {Response}", responseString);
        
        // deserialize the response
        var responseJson = JsonConvert.DeserializeObject(responseString);
        
        return responseJson;
    }

    private static async Task<SensorDataOutput> GetSensorData(string thingName)
    {
        var sensorData = new SensorDataOutput();
        
        try
        {
            // QueryRequest queryRequest = new QueryRequest();
            var query = @"
                    SELECT *
                    FROM (
                        SELECT *
                        FROM ""MoistureSensorTimestreamDB"".""MoistureSensorTable""
                        WHERE deviceId='0506' AND measure_name IN ('moisture', 'pressure', 'temperature')
                        ORDER BY measure_name, time DESC
                    ) t
                        WHERE time IN (
                        SELECT MAX(time)
                        FROM ""MoistureSensorTimestreamDB"".""MoistureSensorTable""
                        WHERE deviceId='" + thingName + @"' AND measure_name IN ('moisture', 'pressure', 'temperature')
                        GROUP BY measure_name
                    )
            ";
            
            var data = await TimestreamFunctions.RunQueryAsync(query);
            var enumerable = data.ToList();
            Logger.LogInformation("Data count: {DataCount}", enumerable.Count());

            foreach (var row in enumerable)
            {
                // remove first and last character
                var str = row[1..^1];
                // split by comma
                var split = str.Split(',');
                // loop through each value until we find string that starts with measure_name
                foreach (var s in split)
                {
                    if (s.StartsWith("measure_name"))
                    {
                        var measureName = s.Split('=')[1];
                        Logger.LogInformation("Measure name: {MeasureName}", measureName);
                        
                        // check if measureName is either temperature, pressure, or moisture
                        if (measureName is "temperature" or "pressure" or "moisture")
                        {
                            foreach (var sAgain in split)
                            {
                                if (sAgain.StartsWith("time=")) // don't want timestamp
                                {
                                    var measureTime = sAgain.Split('=')[1];
                                    Logger.LogInformation("Measure time: {MeasureTime}", measureTime);
                                    
                                    // check if measureName is either temperature, pressure, or moisture
                                    if (measureName is "temperature")
                                    {
                                        sensorData.TemperatureTime = DateTime.Parse(measureTime);
                                    }
                                    else if (measureName is "pressure")
                                    {
                                        sensorData.PressureTime = DateTime.Parse(measureTime);
                                    }
                                    else if (measureName is "moisture")
                                    {
                                        sensorData.MoistureTime = DateTime.Parse(measureTime);
                                    }
                                }
                                
                                if (sAgain.StartsWith("measure_value::bigint"))
                                {
                                    var measureValue = sAgain.Split('=')[1];
                                    Logger.LogInformation("Measure value: {MeasureValue}", measureValue);
                                    
                                    // check if measureName is either temperature, pressure, or moisture
                                    if (measureName is "temperature")
                                    {
                                        sensorData.Temperature = int.Parse(measureValue);
                                    }
                                    else if (measureName is "pressure")
                                    {
                                        sensorData.Pressure = int.Parse(measureValue);
                                    }
                                    else if (measureName is "moisture")
                                    {
                                        sensorData.Moisture = int.Parse(measureValue);
                                    }
                                }
                            }
                        }
                    }
                }
            }
        } catch(Exception e)
        {
            // Some queries might fail with 500 if the result of a sequence function has more than 10000 entries
            Console.WriteLine(e.ToString());
            
            throw new Exception("Failed to get sensor data");
        }

        return sensorData;
    }

    private static async Task<string> UpdateShadow(string thingName, MemoryStream shadow)
    {
        // create a request to update the shadow
        var updateShadowRequest = new UpdateThingShadowRequest
        {
            ThingName = thingName,
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

