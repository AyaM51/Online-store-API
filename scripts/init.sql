CREATE TABLE users (
    Id SERIAL PRIMARY KEY,
    Nickname TEXT NOT NULL UNIQUE,
    Password TEXT NOT NULL,
    Role TEXT DEFAULT 'user'
);

CREATE TABLE products (
    Id SERIAL PRIMARY KEY,
    Name TEXT NOT NULL,
    Manufacturer TEXT NOT NULL,
    Quantity INTEGER NOT NULL,
    Price NUMERIC(10, 2) NOT NULL,
    Description TEXT,
    Available BOOLEAN NOT NULL
);

CREATE TABLE orders (
    Id SERIAL PRIMARY KEY,
    UserId INTEGER NOT NULL,
    Date TIMESTAMP NOT NULL,
    TotalPrice NUMERIC(10, 2),
    Status TEXT,
    CONSTRAINT FK_Orders_Users_UserId FOREIGN KEY (UserId) REFERENCES Users (Id) ON DELETE CASCADE
);

CREATE TABLE categories (
    Id SERIAL PRIMARY KEY,
    Name TEXT NOT NULL,
    ParentId INTEGER
);

CREATE TABLE attributes (
    Id SERIAL PRIMARY KEY,
    Name TEXT NOT NULL
);

CREATE TABLE productsAttributes (
    ProductId INTEGER NOT NULL,
    AttributeId INTEGER NOT NULL,
    Value TEXT NOT NULL,
    CONSTRAINT PK_HasAttributes PRIMARY KEY (ProductId, AttributeId),
    CONSTRAINT FK_ProductsAttributes_Product FOREIGN KEY (ProductId) REFERENCES Products (Id) ON DELETE CASCADE,
    CONSTRAINT FK_ProductsAttributes_Attribute FOREIGN KEY (AttributeId) REFERENCES Attributes (Id) ON DELETE CASCADE
);

CREATE TABLE productsCategories (
    ProductId INTEGER NOT NULL,
    CategoryId INTEGER NOT NULL,
    CONSTRAINT PK_HasCategories PRIMARY KEY (ProductId, CategoryId),
    CONSTRAINT FK_ProductsCategories_Product FOREIGN KEY (ProductId) REFERENCES Products (Id) ON DELETE CASCADE,
    CONSTRAINT FK_ProductsCategories_Category FOREIGN KEY (CategoryId) REFERENCES Categories (Id) ON DELETE CASCADE
);

CREATE TABLE ordersProducts (
    Id SERIAL PRIMARY KEY,
    OrderId INTEGER NOT NULL,
    ProductId INTEGER NOT NULL,
    Quantity INTEGER NOT NULL,
    Price NUMERIC(10, 2) NOT NULL,
    CONSTRAINT FK_OrdersProducts_Orders FOREIGN KEY (OrderId) REFERENCES Orders (Id) ON DELETE CASCADE,
    CONSTRAINT FK_OrdersProducts_Products FOREIGN KEY (ProductId) REFERENCES Products (Id) ON DELETE CASCADE
);



